package lifecycle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
)

// captureHooks は、SupervisedRunner.Register が登録する start / stop 関数を
// 生成 mock（MockRegistrar）経由で捕捉して返します。
func captureHooks(t *testing.T, runner SupervisedRunner) (func(context.Context) error, func(context.Context) error) {
	t.Helper()

	ctrl := gomock.NewController(t)
	reg := mock_lifecycle.NewMockRegistrar(ctrl)
	var start, stop func(context.Context) error
	reg.EXPECT().RegisterStart(gomock.AssignableToTypeOf(start)).
		Do(func(fn func(context.Context) error) { start = fn }).Times(1)
	reg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(stop)).
		Do(func(fn func(context.Context) error) { stop = fn }).Times(1)

	runner.Register(reg)
	return start, stop
}

func TestSupervisedRunner_Register(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("OnStartでBodyを起動しOnStopで実行contextをキャンセルして完了を待つ", func(t *testing.T) {
			t.Parallel()

			var auxStarted, auxStopped bool
			bodyCtxDone := make(chan struct{})
			var bodyCtxErr error

			start, stop := captureHooks(t, SupervisedRunner{
				OnStartAux: func() { auxStarted = true },
				Body: func(ctx context.Context) {
					<-ctx.Done() // OnStop の cancel で解放される
					bodyCtxErr = ctx.Err()
					close(bodyCtxDone)
				},
				OnStopAux: func(context.Context) { auxStopped = true },
			})

			require.NotNil(t, start)
			require.NotNil(t, stop)

			require.NoError(t, start(context.Background()))
			assert.True(t, auxStarted)

			require.NoError(t, stop(context.Background()))
			<-bodyCtxDone
			require.ErrorIs(t, bodyCtxErr, context.Canceled)
			assert.True(t, auxStopped)
		})

		t.Run("起動contextのキャンセルはBodyのcontextへ伝播しない", func(t *testing.T) {
			t.Parallel()

			bodyRan := make(chan struct{})
			var bodyCtxErr error

			start, stop := captureHooks(t, SupervisedRunner{
				Body: func(ctx context.Context) {
					bodyCtxErr = ctx.Err()
					close(bodyRan)
				},
			})

			startCtx, cancel := context.WithCancel(context.Background())
			cancel()
			require.NoError(t, start(startCtx))

			<-bodyRan
			require.NoError(t, bodyCtxErr)

			// 後始末（goroutine リーク防止のため OnStop を呼ぶ）。
			require.NoError(t, stop(context.Background()))
		})

		t.Run("Bodyが既に完了していればOnStopは即座に返る", func(t *testing.T) {
			t.Parallel()

			bodyDone := make(chan struct{})
			start, stop := captureHooks(t, SupervisedRunner{
				Body: func(context.Context) { close(bodyDone) },
			})

			require.NoError(t, start(context.Background()))
			<-bodyDone // Body は即時完了

			require.NoError(t, stop(context.Background()))
		})

		t.Run("Body完了前でも猶予切れ_stopCtx満了_でOnStopは返る", func(t *testing.T) {
			t.Parallel()

			block := make(chan struct{})
			start, stop := captureHooks(t, SupervisedRunner{
				// ctx を無視してブロックし続ける Body（drain が完了しないケース）。
				Body: func(context.Context) { <-block },
			})

			require.NoError(t, start(context.Background()))

			stopCtx, cancel := context.WithCancel(context.Background())
			cancel()
			require.NoError(t, stop(stopCtx))

			close(block) // goroutine の後始末
		})

		t.Run("Bodyとauxがnilでも安全に登録_実行できる", func(t *testing.T) {
			t.Parallel()

			start, stop := captureHooks(t, SupervisedRunner{})

			require.NoError(t, start(context.Background()))
			require.NoError(t, stop(context.Background()))
		})
	})
}
