package outbox

import (
	"context"
	"testing"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	mock_outbox "go-boilerplate/internal/usecase/boundary/outbox/mock"
	"go-boilerplate/internal/usecase/boundary/publisher"
	mock_publisher "go-boilerplate/internal/usecase/boundary/publisher/mock"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

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
		store:       store,
		publisher:   pub,
		metrics:     metrics,
		logging:     logging.NewTestLogger(t),
		tracer:      observability.NewNoopLayerTracer(t),
		maxAttempts: DefaultMaxAttempts,
	}
}

func (*countingMetrics) SetLagSeconds(context.Context, int64) {}
func (m *countingMetrics) IncDead(context.Context)            { m.deadCount++ }

func deliverMessage(t *testing.T) outboxbndry.PendingMessage {
	t.Helper()
	return outboxbndry.PendingMessage{
		ID:        1,
		MessageID: uuid.NewTestFromSalt(t, "msg"),
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

		t.Run("publish 失敗かつ attempts が上限未満なら MarkFailed のみで published=false・error=nil", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := deliverMessage(t)

			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(xerrors.New("publish failed"))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any()).Return(int32(1), nil)

			u := newDeliverUsecase(t, store, pub, observability.NewNoopOutboxMetrics(t))
			published, err := u.deliver(context.Background(), msg)

			require.NoError(t, err)
			assert.False(t, published)
		})

		t.Run("publish 失敗かつ attempts が上限到達なら MarkDead し IncDead を計上して published=false・error=nil", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			metrics := &countingMetrics{}
			msg := deliverMessage(t)

			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(xerrors.New("publish failed"))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any()).Return(DefaultMaxAttempts, nil)
			store.EXPECT().MarkDead(gomock.Any(), msg.ID).Return(nil)

			u := newDeliverUsecase(t, store, pub, metrics)
			published, err := u.deliver(context.Background(), msg)

			require.NoError(t, err)
			assert.False(t, published)
			assert.Equal(t, 1, metrics.deadCount)
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
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any()).Return(int32(0), wantErr)

			u := newDeliverUsecase(t, store, pub, observability.NewNoopOutboxMetrics(t))
			published, err := u.deliver(context.Background(), msg)

			require.ErrorIs(t, err, wantErr)
			assert.False(t, published)
		})

		t.Run("attempts 上限到達で MarkDead のエラーはそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := deliverMessage(t)
			wantErr := xerrors.New("mark dead failed")

			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(xerrors.New("publish failed"))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any()).Return(DefaultMaxAttempts, nil)
			store.EXPECT().MarkDead(gomock.Any(), msg.ID).Return(wantErr)

			u := newDeliverUsecase(t, store, pub, observability.NewNoopOutboxMetrics(t))
			published, err := u.deliver(context.Background(), msg)

			require.ErrorIs(t, err, wantErr)
			assert.False(t, published)
		})
	})
}
