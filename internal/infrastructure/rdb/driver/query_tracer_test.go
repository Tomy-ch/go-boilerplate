package driver

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/xerrors"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

var errQueryForTest = xerrors.New("query failed for test")

type fakeQueryRecorder struct {
	observed int
}

func (f *fakeQueryRecorder) Observe(_ context.Context, _ QueryAttrs) {
	f.observed++
}

func newTestQueryTracer(t *testing.T) (*queryTracer, *mock_logging.MockLogger) {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)

	ctrl := gomock.NewController(t)
	mockLogger := mock_logging.NewMockLogger(ctrl)
	mockLogger.EXPECT().Named(gomock.Any()).Return(mockLogger).AnyTimes()

	lf := logging.NewTestLogFieldBuilder(t)

	qt, ok := NewQueryTracer(dbCfg, obsCfg, otelpgx.NewTracer(), nil, mockLogger, lf).(*queryTracer)
	require.True(t, ok)
	return qt, mockLogger
}

// newObservedQueryTracer は、与えられた logger を使う queryTracer を返します。
// 観測ログ自体は呼び出し側で logging.NewObservedTestLogger から受け取り、
// 終了ログに乗る Field のキー検証に用います（observer 型を本パッケージで直接参照しないため）。
func newObservedQueryTracer(t *testing.T, logger logging.Logger) *queryTracer {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)

	lf := logging.NewTestLogFieldBuilder(t)

	qt, ok := NewQueryTracer(dbCfg, obsCfg, otelpgx.NewTracer(), nil, logger, lf).(*queryTracer)
	require.True(t, ok)
	return qt
}

func TestNewQueryTracer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定値を反映した queryTracer を返す", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			obsCfg := config.NewObservabilityConfig(cfg)

			ctrl := gomock.NewController(t)
			mockLogger := mock_logging.NewMockLogger(ctrl)
			mockLogger.EXPECT().Named(gomock.Any()).Return(mockLogger).AnyTimes()
			lf := logging.NewTestLogFieldBuilder(t)

			// recorder=nil 時（メトリクス記録なし）の動作を検証する。
			qt, ok := NewQueryTracer(dbCfg, obsCfg, otelpgx.NewTracer(), nil, mockLogger, lf).(*queryTracer)
			require.True(t, ok)
			assert.NotNil(t, qt.Tracer)
			assert.Nil(t, qt.recorder)
			assert.Equal(t, obsCfg.MaskedDBQueryArgs(), qt.maskArgs)
			assert.Equal(t, dbCfg.SlowQueryWarnThreshold(), qt.slowThreshold)
		})
	})
}

func Test_queryTracer_TraceQueryEnd(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("開始情報が無い場合はログを出さずに終了する", func(t *testing.T) {
			t.Parallel()

			qt, _ := newTestQueryTracer(t)
			// queryLogKey を含まない context では Error/Warn は呼ばれない（mock 未設定で検知）。
			qt.TraceQueryEnd(context.Background(), nil, pgx.TraceQueryEndData{})
		})

		t.Run("正常終了時は Info ログを出す", func(t *testing.T) {
			t.Parallel()

			qt, mockLogger := newTestQueryTracer(t)
			qt.slowThreshold = time.Second
			mockLogger.EXPECT().Info(gomock.Any(), "DB query completed", gomock.Any()).Times(1)

			ctx := context.WithValue(context.Background(), queryLogKey{}, queryLogData{
				sql:   "SELECT 1",
				start: time.Now(),
			})
			qt.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
		})

		t.Run("正常終了ログに latency_ms と query フィールドが乗る", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			qt := newObservedQueryTracer(t, logger)
			qt.slowThreshold = time.Second

			ctx := context.WithValue(context.Background(), queryLogKey{}, queryLogData{
				sql:   "SELECT 1",
				start: time.Now(),
			})
			qt.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})

			entries := observed.FilterMessage("DB query completed").All()
			require.Len(t, entries, 1)

			ctxMap := entries[0].ContextMap()
			assert.Contains(t, ctxMap, logging.LatencyKey)
			assert.Equal(t, "SELECT 1", ctxMap[logging.RawQueryKey])
			// parentSpanID 未設定（空）なので parent_span_id は出力されない。
			assert.NotContains(t, ctxMap, logging.ParentSpanIDKey)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エラー時は Error ログを出す", func(t *testing.T) {
			t.Parallel()

			qt, mockLogger := newTestQueryTracer(t)
			mockLogger.EXPECT().Error(gomock.Any(), "DB query failed", gomock.Any()).Times(1)

			ctx := context.WithValue(context.Background(), queryLogKey{}, queryLogData{
				sql:   "SELECT 1",
				start: time.Now(),
			})
			qt.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: errQueryForTest})
		})

		t.Run("エラーログに internal_error と latency_ms フィールドが乗る", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			qt := newObservedQueryTracer(t, logger)

			ctx := context.WithValue(context.Background(), queryLogKey{}, queryLogData{
				sql:   "SELECT 1",
				start: time.Now(),
			})
			qt.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: errQueryForTest})

			entries := observed.FilterMessage("DB query failed").All()
			require.Len(t, entries, 1)

			ctxMap := entries[0].ContextMap()
			assert.NotEmpty(t, ctxMap)
			assert.Contains(t, ctxMap, logging.InternalErrorKey)
			assert.Contains(t, ctxMap, logging.LatencyKey)
		})

		t.Run("スロークエリ時は Warn ログを出す", func(t *testing.T) {
			t.Parallel()

			qt, mockLogger := newTestQueryTracer(t)
			qt.slowThreshold = time.Millisecond
			mockLogger.EXPECT().Warn(gomock.Any(), "DB slow query", gomock.Any()).Times(1)

			ctx := context.WithValue(context.Background(), queryLogKey{}, queryLogData{
				sql:   "SELECT 1",
				start: time.Now().Add(-time.Hour),
			})
			qt.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
		})
	})
}

func Test_queryTracer_TraceQueryStart(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("開始時のspanIDがparentSpanIDとして終了処理経路まで引き継がれる", func(t *testing.T) {
			t.Parallel()

			ctx, end := observability.NewStubSpanContext(t)
			defer end()

			wantSpanID := observability.ExtractTraceContext(ctx).SpanID()
			require.NotEmpty(t, wantSpanID)

			qt, mockLogger := newTestQueryTracer(t)
			qt.slowThreshold = time.Second
			mockLogger.EXPECT().Info(gomock.Any(), "DB query completed", gomock.Any()).Times(1)

			// TraceQueryStart が開始時点の spanID を queryLogData.parentSpanID に取り込む。
			startedCtx := qt.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})

			ld, ok := startedCtx.Value(queryLogKey{}).(queryLogData)
			require.True(t, ok)
			assert.Equal(t, wantSpanID, ld.parentSpanID)

			// 手組みの context を使わず TraceQueryStart の出力 context をそのまま
			// TraceQueryEnd へ渡し、parentSpanID が終了処理経路へ引き継がれることを確認する。
			qt.TraceQueryEnd(startedCtx, nil, pgx.TraceQueryEndData{})
		})

		t.Run("正常終了ログに parent_span_id フィールドが乗る", func(t *testing.T) {
			t.Parallel()

			ctx, end := observability.NewStubSpanContext(t)
			defer end()

			// 開始時点の span ID が parent_span_id としてログに乗ることを値レベルで検証する。
			wantParentSpanID := observability.ExtractTraceContext(ctx).SpanID()
			require.NotEmpty(t, wantParentSpanID)

			logger, observed := logging.NewObservedTestLogger(t)
			qt := newObservedQueryTracer(t, logger)
			qt.slowThreshold = time.Second

			startedCtx := qt.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})
			qt.TraceQueryEnd(startedCtx, nil, pgx.TraceQueryEndData{})

			entries := observed.FilterMessage("DB query completed").All()
			require.Len(t, entries, 1)
			assert.Equal(t, wantParentSpanID, entries[0].ContextMap()[logging.ParentSpanIDKey])
		})
	})
}

func Test_queryTracer_endFields_Mask(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("maskArgs が true の場合は args 件数フィールドが付かない", func(t *testing.T) {
			t.Parallel()

			ld := queryLogData{
				sql:   "SELECT $1",
				args:  []any{"secret"},
				start: time.Now(),
			}

			maskedLogger, maskedLogs := logging.NewObservedTestLogger(t)
			qtMasked := newObservedQueryTracer(t, maskedLogger)
			qtMasked.maskArgs = true
			qtMasked.slowThreshold = time.Second
			qtMasked.TraceQueryEnd(
				context.WithValue(context.Background(), queryLogKey{}, ld), nil, pgx.TraceQueryEndData{})

			plainLogger, plainLogs := logging.NewObservedTestLogger(t)
			qtPlain := newObservedQueryTracer(t, plainLogger)
			qtPlain.maskArgs = false
			qtPlain.slowThreshold = time.Second
			qtPlain.TraceQueryEnd(
				context.WithValue(context.Background(), queryLogKey{}, ld), nil, pgx.TraceQueryEndData{})

			maskedEntries := maskedLogs.FilterMessage("DB query completed").All()
			require.Len(t, maskedEntries, 1)
			plainEntries := plainLogs.FilterMessage("DB query completed").All()
			require.Len(t, plainEntries, 1)

			// マスク時は args 件数キーが付かず、非マスク時は付く。
			assert.NotContains(t, maskedEntries[0].ContextMap(), logging.QueryArgsCountKey)
			assert.Contains(t, plainEntries[0].ContextMap(), logging.QueryArgsCountKey)
		})
	})
}

func Test_queryTracer_recordQueryMetric(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("recorderが設定されている場合はObserveが呼ばれる", func(t *testing.T) {
			t.Parallel()

			rec := &fakeQueryRecorder{}
			qt := &queryTracer{recorder: rec}
			qt.recordQueryMetric(context.Background(), "SELECT 1", time.Millisecond, nil)

			assert.Equal(t, 1, rec.observed)
		})

		t.Run("recorderがnilの場合は何もしない", func(t *testing.T) {
			t.Parallel()

			qt := &queryTracer{recorder: nil}
			// nil recorder でも panic せず no-op で終了する。
			qt.recordQueryMetric(context.Background(), "SELECT 1", time.Millisecond, nil)
		})
	})
}

func Test_queryTracer_endFields(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		ld := queryLogData{sql: "SELECT $1", args: []any{"secret"}, start: time.Now()}

		t.Run("maskArgsがfalseの場合はargs件数フィールドを含むフィールド列を返す", func(t *testing.T) {
			t.Parallel()

			qt, _ := newTestQueryTracer(t)
			qt.maskArgs = false
			plain := qt.endFields(ld, time.Millisecond, nil)

			qt.maskArgs = true
			masked := qt.endFields(ld, time.Millisecond, nil)

			// マスク時は args を nil にするため args_count フィールドが 1 つ減る。
			assert.Len(t, masked, len(plain)-1)
		})
	})
}
