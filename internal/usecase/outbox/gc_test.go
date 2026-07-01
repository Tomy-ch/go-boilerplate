package outbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-boilerplate/internal/usecase/boundary/clock/testkit"
	mock_outbox "go-boilerplate/internal/usecase/boundary/outbox/mock"
	"go-boilerplate/internal/usecase/outbox"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewGC(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を渡すと非nilのGCUsecaseを生成する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)

			got := outbox.NewGC(store, testkit.NewMockClock(t, time.Time{}))

			assert.NotNil(t, got)
		})
	})
}

func TestGCUsecase_SweepPublished(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("バッチが満たなくなるまで反復し合計削除件数を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
			cutoff := now.Add(-outbox.DefaultRetention)

			gomock.InOrder(
				store.EXPECT().DeletePublished(gomock.Any(), cutoff, int32(2)).Return(int64(2), nil),
				store.EXPECT().DeletePublished(gomock.Any(), cutoff, int32(2)).Return(int64(1), nil),
			)

			total, err := outbox.NewGC(store, testkit.NewMockClock(t, now)).
				SweepPublished(context.Background(), 2)

			require.NoError(t, err)
			assert.Equal(t, int64(3), total)
		})

		t.Run("batchSize が 0 以下なら既定値を使う", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)

			store.EXPECT().
				DeletePublished(gomock.Any(), gomock.Any(), outbox.DefaultGCBatchSize).
				Return(int64(0), nil)

			total, err := outbox.NewGC(store, testkit.NewMockClock(t, time.Time{})).
				SweepPublished(context.Background(), 0)

			require.NoError(t, err)
			assert.Equal(t, int64(0), total)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("途中の削除失敗はそれまでの件数とエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			wantErr := errors.New("delete failed")

			store.EXPECT().DeletePublished(gomock.Any(), gomock.Any(), int32(5)).Return(int64(0), wantErr)

			total, err := outbox.NewGC(store, testkit.NewMockClock(t, time.Time{})).
				SweepPublished(context.Background(), 5)

			require.ErrorIs(t, err, wantErr)
			assert.Equal(t, int64(0), total)
		})

		t.Run("2バッチ目の削除失敗でもそれまでの累計件数を保持して返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			wantErr := errors.New("delete failed")

			// 1バッチ目は batchSize 件削除（=満杯なので反復継続）、2バッチ目でエラー。
			// total には1バッチ目の件数が累積保持される。
			gomock.InOrder(
				store.EXPECT().DeletePublished(gomock.Any(), gomock.Any(), int32(2)).Return(int64(2), nil),
				store.EXPECT().DeletePublished(gomock.Any(), gomock.Any(), int32(2)).Return(int64(0), wantErr),
			)

			total, err := outbox.NewGC(store, testkit.NewMockClock(t, time.Time{})).
				SweepPublished(context.Background(), 2)

			require.ErrorIs(t, err, wantErr)
			assert.Equal(t, int64(2), total)
		})
	})
}
