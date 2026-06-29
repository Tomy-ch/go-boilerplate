package idempotency_test

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/usecase/boundary/clock/testkit"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	mock_idempotency "go-boilerplate/internal/usecase/boundary/idempotency/mock"
	"go-boilerplate/internal/usecase/idempotency"
	mock_idempotencyuc "go-boilerplate/internal/usecase/idempotency/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGCUsecase_SweepExpired(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("バッチが満たなくなるまで反復し合計削除件数を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

			gomock.InOrder(
				store.EXPECT().DeleteExpired(gomock.Any(), now, int32(2)).Return(int64(2), nil),
				store.EXPECT().DeleteExpired(gomock.Any(), now, int32(2)).Return(int64(1), nil),
			)

			total, err := idempotency.NewGC(store, testkit.NewMockClock(t, now), nil).SweepExpired(context.Background(), 2)

			require.NoError(t, err)
			assert.Equal(t, int64(3), total)
		})

		t.Run("batchSize が 0 以下なら既定値を使う", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			store.EXPECT().
				DeleteExpired(gomock.Any(), gomock.Any(), idempotency.DefaultGCBatchSize).
				Return(int64(0), nil)

			total, err := idempotency.NewGC(store, testkit.NewMockClock(t, time.Time{}), nil).SweepExpired(context.Background(), 0)

			require.NoError(t, err)
			assert.Equal(t, int64(0), total)
		})

		t.Run("metrics が nil でも no-op で削除できる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			store.EXPECT().DeleteExpired(gomock.Any(), gomock.Any(), int32(3)).Return(int64(1), nil)

			total, err := idempotency.NewGC(store, testkit.NewMockClock(t, time.Time{}), nil).SweepExpired(context.Background(), 3)

			require.NoError(t, err)
			assert.Equal(t, int64(1), total)
		})

		t.Run("削除件数が正ならバッチごとに IncExpiredCleanup を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockGCMetrics(ctrl)
			now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

			gomock.InOrder(
				store.EXPECT().DeleteExpired(gomock.Any(), now, int32(2)).Return(int64(2), nil),
				store.EXPECT().DeleteExpired(gomock.Any(), now, int32(2)).Return(int64(1), nil),
			)
			gomock.InOrder(
				metrics.EXPECT().IncExpiredCleanup(gomock.Any(), int64(2)),
				metrics.EXPECT().IncExpiredCleanup(gomock.Any(), int64(1)),
			)

			total, err := idempotency.NewGC(store, testkit.NewMockClock(t, now), metrics).SweepExpired(context.Background(), 2)

			require.NoError(t, err)
			assert.Equal(t, int64(3), total)
		})

		t.Run("削除件数が 0 なら IncExpiredCleanup を計上しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockGCMetrics(ctrl)

			store.EXPECT().
				DeleteExpired(gomock.Any(), gomock.Any(), idempotency.DefaultGCBatchSize).
				Return(int64(0), nil)

			total, err := idempotency.NewGC(store, testkit.NewMockClock(t, time.Time{}), metrics).SweepExpired(context.Background(), 0)

			require.NoError(t, err)
			assert.Equal(t, int64(0), total)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("途中の削除失敗はそれまでの件数とエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			wantErr := idempotencybndry.ErrLockTimeout

			store.EXPECT().DeleteExpired(gomock.Any(), gomock.Any(), int32(5)).Return(int64(0), wantErr)

			total, err := idempotency.NewGC(store, testkit.NewMockClock(t, time.Time{}), nil).SweepExpired(context.Background(), 5)

			require.ErrorIs(t, err, wantErr)
			assert.Equal(t, int64(0), total)
		})

		t.Run("削除失敗時は IncExpiredCleanupFailure を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockGCMetrics(ctrl)
			wantErr := idempotencybndry.ErrLockTimeout

			store.EXPECT().DeleteExpired(gomock.Any(), gomock.Any(), int32(5)).Return(int64(0), wantErr)
			metrics.EXPECT().IncExpiredCleanupFailure(gomock.Any())

			total, err := idempotency.NewGC(store, testkit.NewMockClock(t, time.Time{}), metrics).SweepExpired(context.Background(), 5)

			require.ErrorIs(t, err, wantErr)
			assert.Equal(t, int64(0), total)
		})
	})
}
