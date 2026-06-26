package hook_test

import (
	"context"
	"testing"
	"time"

	outboxengine "go-boilerplate/internal/controller/outbox"
	"go-boilerplate/internal/di/outboxrelay/hook"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	outboxuc "go-boilerplate/internal/usecase/outbox"
	mock_relay "go-boilerplate/internal/usecase/outbox/mock"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fakeRegistrar は、登録された start / stop 関数を捕捉するテスト用の lifecycle.Registrar です。
type fakeRegistrar struct {
	start func(context.Context) error
	stop  func(context.Context) error
}

func (f *fakeRegistrar) RegisterStart(fn func(context.Context) error) { f.start = fn }
func (f *fakeRegistrar) RegisterStop(fn func(context.Context) error)  { f.stop = fn }

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

			reg := &fakeRegistrar{}
			hook.RegisterRelayHooks(reg, engine)
			require.NotNil(t, reg.start)
			require.NotNil(t, reg.stop)

			require.NoError(t, reg.start(context.Background()))
			require.NoError(t, reg.stop(context.Background()))
		})
	})
}
