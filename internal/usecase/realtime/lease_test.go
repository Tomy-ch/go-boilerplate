package realtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
)

func newLeaseKeeper(t *testing.T) (LeaseKeeper, *mock_realtime.MockInstanceLeaseStore, *mock_clock.MockClock) {
	t.Helper()

	ctrl := gomock.NewController(t)
	store := mock_realtime.NewMockInstanceLeaseStore(ctrl)
	clk := mock_clock.NewMockClock(ctrl)

	return NewLeaseKeeper(store, clk, observability.NewNoopTracerFactory(t)), store, clk
}

func TestNewLeaseKeeper(t *testing.T) {
	t.Parallel()

	k, _, _ := newLeaseKeeper(t)
	assert.NotNil(t, k)
}

func Test_leaseKeeper_Beat(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("今を HeartbeatAt に、今 + LeaseExpiry を ExpiresAt に書く", func(t *testing.T) {
			t.Parallel()

			k, store, clk := newLeaseKeeper(t)
			now := time.Date(2026, time.August, 29, 1, 2, 3, 0, time.UTC)
			clk.EXPECT().Now().Return(now)
			store.EXPECT().Heartbeat(gomock.Any(), rt.InstanceLease{
				InstanceID: "inst-1", HeartbeatAt: now, ExpiresAt: now.Add(2 * time.Minute),
			}).Return(nil)

			require.NoError(t, k.Beat(t.Context(), "inst-1"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("store の失敗はそのまま返す", func(t *testing.T) {
			t.Parallel()

			k, store, clk := newLeaseKeeper(t)
			clk.EXPECT().Now().Return(time.Unix(0, 0))
			store.EXPECT().Heartbeat(gomock.Any(), gomock.Any()).Return(apperror.ErrUnavailable)

			require.ErrorIs(t, k.Beat(t.Context(), "inst-1"), apperror.ErrUnavailable)
		})
	})
}

func Test_leaseKeeper_Release(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("記録を削除する", func(t *testing.T) {
			t.Parallel()

			k, store, _ := newLeaseKeeper(t)
			store.EXPECT().Delete(gomock.Any(), rt.InstanceID("inst-1")).Return(nil)

			require.NoError(t, k.Release(t.Context(), "inst-1"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("store の失敗はそのまま返す", func(t *testing.T) {
			t.Parallel()

			k, store, _ := newLeaseKeeper(t)
			store.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(apperror.ErrUnavailable)

			require.ErrorIs(t, k.Release(t.Context(), "inst-1"), apperror.ErrUnavailable)
		})
	})
}
