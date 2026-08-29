package realtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest/observer"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	mock_ucrealtime "go-boilerplate/internal/usecase/realtime/mock"
)

func newHeartbeat(t *testing.T) (*Heartbeat, *mock_ucrealtime.MockLeaseKeeper, *mock_clock.MockSleeper, *observer.ObservedLogs) {
	t.Helper()

	ctrl := gomock.NewController(t)
	keeper := mock_ucrealtime.NewMockLeaseKeeper(ctrl)
	sleeper := mock_clock.NewMockSleeper(ctrl)
	log, logs := logging.NewObservedTestLogger(t)

	return NewHeartbeat(keeper, "inst-1", sleeper, log, observability.NewNoopTracerFactory(t)), keeper, sleeper, logs
}

func TestNewHeartbeat(t *testing.T) {
	t.Parallel()

	h, _, _, _ := newHeartbeat(t)
	assert.NotNil(t, h)
}

func TestHeartbeat_Run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最初の 1 回を即座に書き、LeaseHeartbeatInterval ごとに書き直す", func(t *testing.T) {
			t.Parallel()

			h, keeper, sleeper, _ := newHeartbeat(t)
			gomock.InOrder(
				keeper.EXPECT().Beat(gomock.Any(), rt.InstanceID("inst-1")).Return(nil),
				sleeper.EXPECT().Sleep(gomock.Any(), ucrealtime.LeaseHeartbeatInterval).Return(nil),
				keeper.EXPECT().Beat(gomock.Any(), rt.InstanceID("inst-1")).Return(nil),
				sleeper.EXPECT().Sleep(gomock.Any(), ucrealtime.LeaseHeartbeatInterval).Return(context.Canceled),
			)

			require.NoError(t, h.Run(t.Context()))
		})

		t.Run("開始時点で ctx が完了していれば書かずに nil を返す", func(t *testing.T) {
			t.Parallel()

			h, _, _, _ := newHeartbeat(t)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			require.NoError(t, h.Run(ctx))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("1 回の失敗は記録して次の周期へ続ける", func(t *testing.T) {
			t.Parallel()

			h, keeper, sleeper, logs := newHeartbeat(t)
			gomock.InOrder(
				keeper.EXPECT().Beat(gomock.Any(), gomock.Any()).Return(apperror.ErrUnavailable),
				sleeper.EXPECT().Sleep(gomock.Any(), ucrealtime.LeaseHeartbeatInterval).Return(nil),
				keeper.EXPECT().Beat(gomock.Any(), gomock.Any()).Return(nil),
				sleeper.EXPECT().Sleep(gomock.Any(), ucrealtime.LeaseHeartbeatInterval).Return(context.Canceled),
			)

			require.NoError(t, h.Run(t.Context()))
			assert.Equal(t, 1, logs.FilterMessage("failed to heartbeat instance lease").Len())
		})

		t.Run("ctx 完了による失敗は記録しない", func(t *testing.T) {
			t.Parallel()

			h, keeper, _, logs := newHeartbeat(t)
			ctx, cancel := context.WithCancel(t.Context())
			keeper.EXPECT().Beat(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, rt.InstanceID) error {
				cancel()

				return apperror.ErrCanceled
			})

			require.NoError(t, h.Run(ctx))
			assert.Equal(t, 0, logs.Len())
		})
	})
}
