package driver

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

var errQueryForTest = xerrors.New("query failed for test")

func newTestQueryTracer(t *testing.T) (*queryTracer, *mock_logging.MockLogger) {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)

	ctrl := gomock.NewController(t)
	mockLogger := mock_logging.NewMockLogger(ctrl)
	mockLogger.EXPECT().Named(gomock.Any()).Return(mockLogger).AnyTimes()

	lf := logging.NewTestLogFieldBuilder(t)

	qt, ok := NewQueryTracer(dbCfg, obsCfg, otelpgx.NewTracer(), mockLogger, lf).(*queryTracer)
	require.True(t, ok)
	return qt, mockLogger
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
			lf := logging.NewTestLogFieldBuilder(t)

			qt, ok := NewQueryTracer(dbCfg, obsCfg, otelpgx.NewTracer(), mockLogger, lf).(*queryTracer)
			require.True(t, ok)
			assert.NotNil(t, qt.Tracer)
			assert.Equal(t, obsCfg.MaskedDBQueryArgs(), qt.maskArgs)
			assert.Equal(t, dbCfg.SlowQueryWarnThreshold(), qt.slowThreshold)
		})
	})
}

func TestQueryTracer_TraceQueryEnd(t *testing.T) {
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
			mockLogger.EXPECT().Info("DB query completed", gomock.Any()).Times(1)

			ctx := context.WithValue(context.Background(), queryLogKey{}, queryLogData{
				sql:   "SELECT 1",
				start: time.Now(),
			})
			qt.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エラー時は Error ログを出す", func(t *testing.T) {
			t.Parallel()

			qt, mockLogger := newTestQueryTracer(t)
			mockLogger.EXPECT().Error("DB query failed", gomock.Any()).Times(1)

			ctx := context.WithValue(context.Background(), queryLogKey{}, queryLogData{
				sql:   "SELECT 1",
				start: time.Now(),
			})
			qt.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: errQueryForTest})
		})

		t.Run("スロークエリ時は Warn ログを出す", func(t *testing.T) {
			t.Parallel()

			qt, mockLogger := newTestQueryTracer(t)
			qt.slowThreshold = time.Millisecond
			mockLogger.EXPECT().Warn("DB slow query", gomock.Any()).Times(1)

			ctx := context.WithValue(context.Background(), queryLogKey{}, queryLogData{
				sql:   "SELECT 1",
				start: time.Now().Add(-time.Hour),
			})
			qt.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
		})
	})
}

func TestQueryTracer_endFields_Mask(t *testing.T) {
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

			qtMasked, _ := newTestQueryTracer(t)
			qtMasked.maskArgs = true
			masked := qtMasked.endFields(context.Background(), ld, time.Millisecond, nil)

			qtPlain, _ := newTestQueryTracer(t)
			qtPlain.maskArgs = false
			plain := qtPlain.endFields(context.Background(), ld, time.Millisecond, nil)

			// マスク時は args 件数フィールドが付かないぶん、フィールド数が少なくなる。
			assert.Less(t, len(masked), len(plain))
		})
	})
}

func TestNewTracedDB_RealQueryInstrumentation(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)

	ctrl := gomock.NewController(t)
	mockLogger := mock_logging.NewMockLogger(ctrl)
	mockLogger.EXPECT().Named(gomock.Any()).Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().Info("DB query completed", gomock.Any()).MinTimes(1)
	mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).MinTimes(1)
	lf := logging.NewTestLogFieldBuilder(t)

	tracer := NewQueryTracer(dbCfg, obsCfg, otelpgx.NewTracer(), mockLogger, lf)

	db, err := NewTracedDB(dbCfg, osCfg, dbConnCfg, tracer)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// 正常クエリは TraceQueryEnd(Info) のログが出る。
	_, err = db.Exec(ctx, "SELECT 1")
	require.NoError(t, err)

	// 失敗クエリは終了で Error ログが出る。
	_, err = db.Exec(ctx, "SELECT 1 FROM no_such_table_for_test")
	require.Error(t, err)
}
