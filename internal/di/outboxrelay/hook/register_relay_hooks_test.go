package hook_test

import (
	"context"
	"testing"
	"time"

	outboxengine "go-boilerplate/internal/controller/outbox"
	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	"go-boilerplate/internal/di/outboxrelay/hook"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	outboxuc "go-boilerplate/internal/usecase/outbox"
	mock_relay "go-boilerplate/internal/usecase/outbox/mock"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRegisterRelayHooks(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("OnStart で poll ループを起動し OnStop で停止する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_relay.NewMockRelayUsecase(ctrl)
			sleeper := mock_clock.NewMockSleeper(ctrl)

			// pending は常に空。Sleep は ctx 完了（OnStop の cancel）まで待機する。
			uc.EXPECT().RelayBatch(gomock.Any(), gomock.Any()).Return(outboxuc.RelayResult{}, nil).AnyTimes()
			uc.EXPECT().RecordLag(gomock.Any()).Return(nil).AnyTimes()
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, _ time.Duration) error {
					<-ctx.Done()
					return ctx.Err()
				}).AnyTimes()

			engine := outboxengine.NewEngine(uc, sleeper, logging.NewTestLogger(t),
				observability.NewNoopTracerFactory(t),
				outboxengine.Settings{BatchSize: 100, PollInterval: time.Second, ErrorBackoff: time.Second})

			// RegisterRelayHooks が登録する start / stop 関数を生成 mock 経由で捕捉する。
			var start, stop func(context.Context) error
			reg := mock_lifecycle.NewMockRegistrar(ctrl)
			reg.EXPECT().RegisterStart(gomock.AssignableToTypeOf(start)).
				Do(func(fn func(context.Context) error) { start = fn }).Times(1)
			reg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(stop)).
				Do(func(fn func(context.Context) error) { stop = fn }).Times(1)

			hook.RegisterRelayHooks(reg, engine)
			require.NotNil(t, start)
			require.NotNil(t, stop)

			require.NoError(t, start(context.Background()))
			require.NoError(t, stop(context.Background()))
		})
	})
}
