package realtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	mock_ucrealtime "go-boilerplate/internal/usecase/realtime/mock"
)

// heartbeatMocks は、heartbeat loop の依存の test double と、観測したログを数える関数です。
type heartbeatMocks struct {
	keeper  *mock_ucrealtime.MockLeaseKeeper
	sleeper *mock_clock.MockSleeper
	// logCount は、msg のログの件数を返します（msg が空なら全件）。
	logCount func(msg string) int
}

func newHeartbeat(t *testing.T) (*Heartbeat, heartbeatMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := heartbeatMocks{keeper: mock_ucrealtime.NewMockLeaseKeeper(ctrl), sleeper: mock_clock.NewMockSleeper(ctrl)}
	log, logs := logging.NewObservedTestLogger(t)
	m.logCount = func(msg string) int {
		if msg == "" {
			return logs.Len()
		}

		return logs.FilterMessage(msg).Len()
	}

	return NewHeartbeat(m.keeper, "inst-1", m.sleeper, log,
		observability.NewNoopTracerFactory(t), observability.NewNoopRealtimeMetrics(t)), m
}

func TestNewHeartbeat(t *testing.T) {
	t.Parallel()

	h, _ := newHeartbeat(t)
	assert.NotNil(t, h)
}

func TestHeartbeat_Run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最初の 1 回を即座に書き、LeaseHeartbeatInterval ごとに書き直す", func(t *testing.T) {
			t.Parallel()

			h, m := newHeartbeat(t)
			gomock.InOrder(
				m.keeper.EXPECT().Beat(gomock.Any(), rt.InstanceID("inst-1")).Return(nil),
				m.sleeper.EXPECT().Sleep(gomock.Any(), ucrealtime.LeaseHeartbeatInterval).Return(nil),
				m.keeper.EXPECT().Beat(gomock.Any(), rt.InstanceID("inst-1")).Return(nil),
				m.sleeper.EXPECT().Sleep(gomock.Any(), ucrealtime.LeaseHeartbeatInterval).Return(context.Canceled),
			)

			require.NoError(t, h.Run(t.Context()))
		})

		t.Run("開始時点で ctx が完了していれば書かずに nil を返す", func(t *testing.T) {
			t.Parallel()

			h, _ := newHeartbeat(t)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			require.NoError(t, h.Run(ctx))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("1 回の失敗は記録して次の周期へ続ける", func(t *testing.T) {
			t.Parallel()

			h, m := newHeartbeat(t)
			gomock.InOrder(
				m.keeper.EXPECT().Beat(gomock.Any(), gomock.Any()).Return(apperror.ErrUnavailable),
				m.sleeper.EXPECT().Sleep(gomock.Any(), ucrealtime.LeaseHeartbeatInterval).Return(nil),
				m.keeper.EXPECT().Beat(gomock.Any(), gomock.Any()).Return(nil),
				m.sleeper.EXPECT().Sleep(gomock.Any(), ucrealtime.LeaseHeartbeatInterval).Return(context.Canceled),
			)

			require.NoError(t, h.Run(t.Context()))
			assert.Equal(t, 1, m.logCount("failed to heartbeat instance lease"))
		})

		t.Run("ctx 完了による失敗は記録しない", func(t *testing.T) {
			t.Parallel()

			h, m := newHeartbeat(t)
			ctx, cancel := context.WithCancel(t.Context())
			m.keeper.EXPECT().Beat(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, rt.InstanceID) error {
				cancel()

				return apperror.ErrCanceled
			})

			require.NoError(t, h.Run(ctx))
			assert.Equal(t, 0, m.logCount(""))
		})
	})
}
