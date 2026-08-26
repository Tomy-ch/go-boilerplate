package outbox

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	mock_relay "go-boilerplate/internal/usecase/outbox/mock"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newWaitDoneEngine(t *testing.T, sleeper *mock_clock.MockSleeper) *Engine {
	t.Helper()
	ctrl := gomock.NewController(t)
	uc := mock_relay.NewMockRelayUsecase(ctrl)
	return NewEngine(uc, sleeper, logging.NewTestLogger(t), observability.NewNoopTracerFactory(t),
		Settings{BatchSize: 100, PollInterval: time.Second, ErrorBackoff: 5 * time.Second})
}

func TestEngine_waitDone(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定時間の待機を完了したら false を返しループを継続させる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			sleeper.EXPECT().Sleep(gomock.Any(), 3*time.Second).Return(nil)

			assert.False(t, newWaitDoneEngine(t, sleeper).waitDone(context.Background(), 3*time.Second))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctx 完了で待機が中断されたら true を返しループを終わらせる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			sleeper.EXPECT().Sleep(gomock.Any(), 3*time.Second).Return(context.Canceled)

			assert.True(t, newWaitDoneEngine(t, sleeper).waitDone(context.Background(), 3*time.Second))
		})
	})
}
