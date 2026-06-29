package outbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock/testkit"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	mock_outbox "go-boilerplate/internal/usecase/boundary/outbox/mock"
	"go-boilerplate/internal/usecase/boundary/publisher"
	mock_publisher "go-boilerplate/internal/usecase/boundary/publisher/mock"
	"go-boilerplate/internal/usecase/boundary/tx"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/outbox"
	mock_relay "go-boilerplate/internal/usecase/outbox/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// passthroughManager は、fn をそのまま実行する tx.Manager mock を返します。
func passthroughManager(t *testing.T, ctrl *gomock.Controller) tx.Manager {
	t.Helper()
	m := mock_tx.NewMockManager(ctrl)
	m.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Times(1)
	return m
}

func newRelayWithMetrics(
	t *testing.T, txm tx.Manager, store outboxbndry.Store, pub publisher.Publisher, metrics outbox.Metrics,
) outbox.RelayUsecase {
	t.Helper()
	return outbox.NewRelay(txm, store, pub, metrics, testkit.NewMockClock(t, time.Time{}),
		logging.NewTestLogger(t), observability.NewNoopTracerFactory(t))
}

func newRelay(t *testing.T, txm tx.Manager, store outboxbndry.Store, pub publisher.Publisher) outbox.RelayUsecase {
	t.Helper()
	return newRelayWithMetrics(t, txm, store, pub, observability.NewNoopOutboxMetrics(t))
}

func pendingMessage(t *testing.T) outboxbndry.PendingMessage {
	t.Helper()
	return outboxbndry.PendingMessage{
		ID:        1,
		MessageID: uuid.NewTestFromSalt(t, "msg"),
		EventType: "e.v1",
		Payload:   []byte(`{"v":1}`),
	}
}

func TestRelayUsecase_RelayBatch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pending が無い場合は publish せず 0 を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)

			store.EXPECT().ClaimPending(gomock.Any(), int32(100)).Return(nil, nil)

			got, err := newRelay(t, passthroughManager(t, ctrl), store, pub).
				RelayBatch(context.Background(), 100)

			require.NoError(t, err)
			assert.Equal(t, 0, got.Claimed)
			assert.Equal(t, 0, got.Published)
		})

		t.Run("publish 成功で MarkPublished し claim 件数を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := pendingMessage(t)

			store.EXPECT().ClaimPending(gomock.Any(), int32(100)).Return([]outboxbndry.PendingMessage{msg}, nil)
			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, m publisher.Message) error {
					assert.Equal(t, msg.MessageID, m.MessageID)
					assert.Equal(t, msg.EventType, m.EventType)
					assert.Equal(t, msg.Payload, m.Payload)
					return nil
				})
			store.EXPECT().MarkPublished(gomock.Any(), msg.ID).Return(nil)

			got, err := newRelay(t, passthroughManager(t, ctrl), store, pub).
				RelayBatch(context.Background(), 100)

			require.NoError(t, err)
			assert.Equal(t, 1, got.Claimed)
			assert.Equal(t, 1, got.Published)
		})

		t.Run("batchSize が 0 以下なら既定値で claim する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)

			store.EXPECT().ClaimPending(gomock.Any(), outbox.DefaultBatchSize).Return(nil, nil)

			_, err := newRelay(t, passthroughManager(t, ctrl), store, pub).
				RelayBatch(context.Background(), 0)

			require.NoError(t, err)
		})

		t.Run("publish 失敗かつ attempts が上限未満なら MarkFailed のみ行い dead 化しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := pendingMessage(t)

			store.EXPECT().ClaimPending(gomock.Any(), int32(100)).Return([]outboxbndry.PendingMessage{msg}, nil)
			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(errors.New("publish failed"))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any()).Return(int32(1), nil)

			got, err := newRelay(t, passthroughManager(t, ctrl), store, pub).
				RelayBatch(context.Background(), 100)

			require.NoError(t, err)
			assert.Equal(t, 1, got.Claimed)
			assert.Equal(t, 0, got.Published)
		})

		t.Run("保存済みヘッダ JSON を復元して publish へ渡す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := pendingMessage(t)
			msg.Headers = []byte(`{"traceparent":"00-x"}`)

			store.EXPECT().ClaimPending(gomock.Any(), int32(100)).Return([]outboxbndry.PendingMessage{msg}, nil)
			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, m publisher.Message) error {
					assert.Equal(t, map[string]string{"traceparent": "00-x"}, m.Headers)
					return nil
				})
			store.EXPECT().MarkPublished(gomock.Any(), msg.ID).Return(nil)

			_, err := newRelay(t, passthroughManager(t, ctrl), store, pub).
				RelayBatch(context.Background(), 100)

			require.NoError(t, err)
		})

		t.Run("ヘッダ JSON が壊れていてもヘッダ無しで publish を継続する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := pendingMessage(t)
			msg.Headers = []byte(`not json`)

			store.EXPECT().ClaimPending(gomock.Any(), int32(100)).Return([]outboxbndry.PendingMessage{msg}, nil)
			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, m publisher.Message) error {
					assert.Nil(t, m.Headers)
					return nil
				})
			store.EXPECT().MarkPublished(gomock.Any(), msg.ID).Return(nil)

			_, err := newRelay(t, passthroughManager(t, ctrl), store, pub).
				RelayBatch(context.Background(), 100)

			require.NoError(t, err)
		})

		t.Run("publish 失敗かつ attempts が上限到達なら MarkDead する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := pendingMessage(t)

			store.EXPECT().ClaimPending(gomock.Any(), int32(100)).Return([]outboxbndry.PendingMessage{msg}, nil)
			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(errors.New("publish failed"))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any()).Return(outbox.DefaultMaxAttempts, nil)
			store.EXPECT().MarkDead(gomock.Any(), msg.ID).Return(nil)
			metrics := mock_relay.NewMockMetrics(ctrl)
			metrics.EXPECT().IncDead(gomock.Any()).Times(1)

			got, err := newRelayWithMetrics(t, passthroughManager(t, ctrl), store, pub, metrics).
				RelayBatch(context.Background(), 100)

			require.NoError(t, err)
			assert.Equal(t, 1, got.Claimed)
			assert.Equal(t, 0, got.Published)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ClaimPending のエラーは 0 件とエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			wantErr := errors.New("claim failed")

			store.EXPECT().ClaimPending(gomock.Any(), int32(100)).Return(nil, wantErr)

			got, err := newRelay(t, passthroughManager(t, ctrl), store, pub).
				RelayBatch(context.Background(), 100)

			require.ErrorIs(t, err, wantErr)
			assert.Equal(t, 0, got.Claimed)
		})

		t.Run("MarkPublished のエラーは tx を巻き戻すエラーとして返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := pendingMessage(t)
			wantErr := errors.New("mark failed")

			store.EXPECT().ClaimPending(gomock.Any(), int32(100)).Return([]outboxbndry.PendingMessage{msg}, nil)
			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
			store.EXPECT().MarkPublished(gomock.Any(), msg.ID).Return(wantErr)

			_, err := newRelay(t, passthroughManager(t, ctrl), store, pub).
				RelayBatch(context.Background(), 100)

			require.ErrorIs(t, err, wantErr)
		})

		t.Run("MarkDead のエラーは tx を巻き戻すエラーとして返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := pendingMessage(t)
			wantErr := errors.New("dead failed")

			store.EXPECT().ClaimPending(gomock.Any(), int32(100)).Return([]outboxbndry.PendingMessage{msg}, nil)
			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(errors.New("publish failed"))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any()).Return(outbox.DefaultMaxAttempts, nil)
			store.EXPECT().MarkDead(gomock.Any(), msg.ID).Return(wantErr)

			_, err := newRelay(t, passthroughManager(t, ctrl), store, pub).
				RelayBatch(context.Background(), 100)

			require.ErrorIs(t, err, wantErr)
		})

		t.Run("複数メッセージ混在時に2件目が失敗すると tx を巻き戻し結果は破棄される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			wantErr := errors.New("mark failed")

			// 1件目は publish 成功で MarkPublished も成功するが、2件目の MarkPublished が
			// DB エラーになる。deliver の DB マーク失敗は tx を巻き戻すエラーなので、
			// 1件目の成功も含めて RelayResult は破棄される（DoWithResult が zero 値を返す）。
			msg1 := pendingMessage(t)
			msg2 := pendingMessage(t)
			msg2.ID = 2

			store.EXPECT().ClaimPending(gomock.Any(), int32(100)).
				Return([]outboxbndry.PendingMessage{msg1, msg2}, nil)
			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil).Times(2)
			store.EXPECT().MarkPublished(gomock.Any(), msg1.ID).Return(nil)
			store.EXPECT().MarkPublished(gomock.Any(), msg2.ID).Return(wantErr)

			got, err := newRelay(t, passthroughManager(t, ctrl), store, pub).
				RelayBatch(context.Background(), 100)

			require.ErrorIs(t, err, wantErr)
			assert.Equal(t, 0, got.Claimed)
			assert.Equal(t, 0, got.Published)
		})

		t.Run("MarkFailed のエラーは tx を巻き戻すエラーとして返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			pub := mock_publisher.NewMockPublisher(ctrl)
			msg := pendingMessage(t)
			wantErr := errors.New("mark failed")

			store.EXPECT().ClaimPending(gomock.Any(), int32(100)).Return([]outboxbndry.PendingMessage{msg}, nil)
			pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(errors.New("publish failed"))
			store.EXPECT().MarkFailed(gomock.Any(), msg.ID, gomock.Any()).Return(int32(0), wantErr)

			_, err := newRelay(t, passthroughManager(t, ctrl), store, pub).
				RelayBatch(context.Background(), 100)

			require.ErrorIs(t, err, wantErr)
		})
	})
}

func TestRelayUsecase_RecordLag(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	build := func(t *testing.T, store outboxbndry.Store, metrics outbox.Metrics) outbox.RelayUsecase {
		t.Helper()
		ctrl := gomock.NewController(t)
		return outbox.NewRelay(
			mock_tx.NewMockManager(ctrl), store, mock_publisher.NewMockPublisher(ctrl),
			metrics, testkit.NewMockClock(t, now),
			logging.NewTestLogger(t), observability.NewNoopTracerFactory(t))
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pending 行があれば経過秒数を lag として記録し nil を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			store.EXPECT().OldestPendingCreatedAt(gomock.Any()).Return(now.Add(-time.Minute), true, nil)
			metrics := mock_relay.NewMockMetrics(ctrl)
			metrics.EXPECT().SetLagSeconds(gomock.Any(), int64(60)).Times(1)

			require.NoError(t, build(t, store, metrics).RecordLag(context.Background()))
		})

		t.Run("pending 行が無ければ 0 を記録して nil を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			store.EXPECT().OldestPendingCreatedAt(gomock.Any()).Return(time.Time{}, false, nil)
			metrics := mock_relay.NewMockMetrics(ctrl)
			metrics.EXPECT().SetLagSeconds(gomock.Any(), int64(0)).Times(1)

			require.NoError(t, build(t, store, metrics).RecordLag(context.Background()))
		})

		t.Run("now より未来の created_at は負の lag を 0 にクランプする", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			store.EXPECT().OldestPendingCreatedAt(gomock.Any()).Return(now.Add(time.Minute), true, nil)
			metrics := mock_relay.NewMockMetrics(ctrl)
			metrics.EXPECT().SetLagSeconds(gomock.Any(), int64(0)).Times(1)

			require.NoError(t, build(t, store, metrics).RecordLag(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("OldestPendingCreatedAt のエラーを伝播し lag は記録しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			wantErr := errors.New("lag query failed")
			store.EXPECT().OldestPendingCreatedAt(gomock.Any()).Return(time.Time{}, false, wantErr)
			// エラー時は SetLagSeconds を呼ばない。
			metrics := mock_relay.NewMockMetrics(ctrl)

			require.ErrorIs(t, build(t, store, metrics).RecordLag(context.Background()), wantErr)
		})
	})
}
