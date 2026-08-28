package outbox

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	mock_outbox "go-boilerplate/internal/usecase/boundary/outbox/mock"
	"go-boilerplate/internal/usecase/boundary/publisher"
	mock_publisher "go-boilerplate/internal/usecase/boundary/publisher/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// deliverNow は、deliver テストで clock が返す固定時刻です。
var deliverNow = time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)

// countingMetrics は、IncDead の呼び出し回数を記録する Metrics スタブです。
type countingMetrics struct{ deadCount int }

func Test_decodeHeaders(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効なヘッダ JSON の場合、map へ復元する", func(t *testing.T) {
			t.Parallel()

			actual := decodeHeaders([]byte(`{"k1":"v1","k2":"v2"}`))
			assert.Equal(t, map[string]string{"k1": "v1", "k2": "v2"}, actual)
		})

		t.Run("空バイト列の場合、nil を返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, decodeHeaders([]byte{}))
		})

		t.Run("nil の場合、nil を返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, decodeHeaders(nil))
		})

		t.Run("壊れた JSON の場合、publish を止めないため nil を返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, decodeHeaders([]byte(`not-json`)))
		})
	})
}

// newDeliverUsecase は、deliver 直接テスト用に mock 依存で relayUsecase を構築します。
func newDeliverUsecase(
	t *testing.T, store outboxbndry.Store, pub publisher.Publisher, metrics Metrics,
) *relayUsecase {
	t.Helper()
	return &relayUsecase{
		store:     store,
		publisher: pub,
		metrics:   metrics,
		clock:     newFixedClock(deliverNow),
		logging:   logging.NewTestLogger(t),
		tracer:    observability.NewNoopLayerTracer(t),
		channel:   outboxbndry.ChannelHTTP,
	}
}

// newFixedClock は、常に at を返す clock.Clock を生成します。
func newFixedClock(at time.Time) clock.Clock {
	return clocktestkit.NewStepClock(at, 0)
}

func (*countingMetrics) SetLagSeconds(context.Context, string, int64)     {}
func (m *countingMetrics) IncDead(context.Context, string)                { m.deadCount++ }
func (*countingMetrics) SetBlockedStreams(context.Context, string, int64) {}

func deliverMessage(t *testing.T) outboxbndry.PendingMessage {
	t.Helper()
	return outboxbndry.PendingMessage{
		ID:        1,
		MessageID: uuidtestkit.NewTestFromSalt(t, "msg"),
		EventType: "e.v1",
		Payload:   []byte(`{"v":1}`),
	}
}

func Test_relayUsecase_deliver(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("publish 成功なら MarkPublished し published=true を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := deliverMessage(t)

			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, m publisher.Message) error {
					assert.Equal(t, msg.MessageID, m.MessageID)
					assert.Equal(t, msg.EventType, m.EventType)
					assert.Equal(t, msg.Payload, m.Payload)
					return nil
				})
			store.EXPECT().MarkPublished(gomock.Any(), msg.ID).Return(nil)

			u := newDeliverUsecase(t, store, pub, observability.NewNoopOutboxMetrics(t))
			published, err := u.deliver(context.Background(), msg)

			require.NoError(t, err)
			assert.True(t, published)
		})

		t.Run("一時失敗なら dead にせず次回試行時刻を進めて published=false・error=nil", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			metrics := &countingMetrics{}
			msg := deliverMessage(t)

			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).
				Return(xerrors.Join(apperror.ErrRetryable, xerrors.New("publish failed")))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ int64, _ string, nextAttemptAt time.Time) error {
					// バックオフは full jitter なので、初回失敗の待機幅 [0, 1s] に収まることを表明する。
					assert.False(t, nextAttemptAt.Before(deliverNow))
					assert.False(t, nextAttemptAt.After(deliverNow.Add(retryInitialInterval)))
					return nil
				})

			u := newDeliverUsecase(t, store, pub, metrics)
			published, err := u.deliver(context.Background(), msg)

			require.NoError(t, err)
			assert.False(t, published)
			assert.Equal(t, 0, metrics.deadCount)
		})

		t.Run("恒久失敗なら理由を残してから MarkDead し IncDead を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			metrics := &countingMetrics{}
			msg := deliverMessage(t)

			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).
				Return(xerrors.Join(apperror.ErrPermanent, xerrors.New("rejected")))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any(), deliverNow).Return(nil)
			store.EXPECT().MarkDead(gomock.Any(), msg.ID).Return(nil)

			u := newDeliverUsecase(t, store, pub, metrics)
			published, err := u.deliver(context.Background(), msg)

			require.NoError(t, err)
			assert.False(t, published)
			assert.Equal(t, 1, metrics.deadCount)
		})

		t.Run("一時失敗の待機は毎回ゼロではない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := deliverMessage(t)

			// full jitter は [0, d] の一様乱数なので、1 回の呼び出しでは「待機 0」と
			// 「バックオフの加算そのものが無い」を区別できない。繰り返して 1 度でも
			// 前進すれば、加算を落とす退行を捕まえられる。
			const attempts = 32
			var advanced bool

			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).
				Return(xerrors.Join(apperror.ErrRetryable, xerrors.New("publish failed"))).Times(attempts)
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ int64, _ string, nextAttemptAt time.Time) error {
					if nextAttemptAt.After(deliverNow) {
						advanced = true
					}
					return nil
				}).Times(attempts)

			u := newDeliverUsecase(t, store, pub, observability.NewNoopOutboxMetrics(t))
			for range attempts {
				_, err := u.deliver(context.Background(), msg)
				require.NoError(t, err)
			}

			assert.True(t, advanced, "一時失敗は次回試行時刻を前進させる")
		})

		t.Run("分類の無い失敗は一時失敗として扱い dead にしない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			metrics := &countingMetrics{}
			msg := deliverMessage(t)

			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(xerrors.New("publish failed"))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any(), gomock.Any()).Return(nil)

			u := newDeliverUsecase(t, store, pub, metrics)
			published, err := u.deliver(context.Background(), msg)

			require.NoError(t, err)
			assert.False(t, published)
			assert.Equal(t, 0, metrics.deadCount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MarkPublished のエラーはそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := deliverMessage(t)
			wantErr := xerrors.New("mark published failed")

			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
			store.EXPECT().MarkPublished(gomock.Any(), msg.ID).Return(wantErr)

			u := newDeliverUsecase(t, store, pub, observability.NewNoopOutboxMetrics(t))
			published, err := u.deliver(context.Background(), msg)

			require.ErrorIs(t, err, wantErr)
			// publish 自体は成功しているため published=true のまま MarkPublished のエラーを返す
			assert.True(t, published)
		})

		t.Run("MarkFailed のエラーはそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := deliverMessage(t)
			wantErr := xerrors.New("mark failed")

			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(xerrors.New("publish failed"))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any(), gomock.Any()).Return(wantErr)

			u := newDeliverUsecase(t, store, pub, observability.NewNoopOutboxMetrics(t))
			published, err := u.deliver(context.Background(), msg)

			require.ErrorIs(t, err, wantErr)
			assert.False(t, published)
		})

		t.Run("恒久失敗で MarkDead のエラーはそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := deliverMessage(t)
			wantErr := xerrors.New("mark dead failed")

			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).
				Return(xerrors.Join(apperror.ErrPermanent, xerrors.New("rejected")))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any(), deliverNow).Return(nil)
			store.EXPECT().MarkDead(gomock.Any(), msg.ID).Return(wantErr)

			u := newDeliverUsecase(t, store, pub, observability.NewNoopOutboxMetrics(t))
			published, err := u.deliver(context.Background(), msg)

			require.ErrorIs(t, err, wantErr)
			assert.False(t, published)
		})
	})
}

func Test_isPermanent(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ErrPermanent を運ぶエラーは恒久失敗と判定する", func(t *testing.T) {
			t.Parallel()

			assert.True(t, isPermanent(xerrors.Join(apperror.ErrPermanent, xerrors.New("rejected"))))
		})

		t.Run("ErrRetryable を運ぶエラーは恒久失敗と判定しない", func(t *testing.T) {
			t.Parallel()

			assert.False(t, isPermanent(xerrors.Join(apperror.ErrRetryable, xerrors.New("unavailable"))))
		})

		t.Run("分類を運ばないエラーは恒久失敗と判定しない", func(t *testing.T) {
			t.Parallel()

			assert.False(t, isPermanent(xerrors.New("unclassified")))
		})
	})
}

func Test_retryDelay(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("初回失敗の待機は初期間隔を超えない", func(t *testing.T) {
			t.Parallel()

			d := retryDelay(0)
			assert.GreaterOrEqual(t, d, time.Duration(0))
			assert.LessOrEqual(t, d, retryInitialInterval)
		})

		t.Run("失敗を重ねても待機は上限を超えない", func(t *testing.T) {
			t.Parallel()

			d := retryDelay(100)
			assert.GreaterOrEqual(t, d, time.Duration(0))
			assert.LessOrEqual(t, d, retryMaxInterval)
		})
	})
}
