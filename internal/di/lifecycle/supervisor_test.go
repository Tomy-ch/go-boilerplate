package lifecycle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRegistrar は、登録された start / stop 関数を捕捉するテスト用の Registrar です。
type fakeRegistrar struct {
	start func(context.Context) error
	stop  func(context.Context) error
}

func (f *fakeRegistrar) RegisterStart(fn func(context.Context) error) { f.start = fn }
func (f *fakeRegistrar) RegisterStop(fn func(context.Context) error)  { f.stop = fn }

func TestSupervisedRunner_Register(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("OnStartでBodyを起動しOnStopで実行contextをキャンセルして完了を待つ", func(t *testing.T) {
			t.Parallel()

			var auxStarted, auxStopped bool
			bodyCtxDone := make(chan struct{})
			var bodyCtxErr error

			reg := &fakeRegistrar{}
			SupervisedRunner{
				OnStartAux: func() { auxStarted = true },
				Body: func(ctx context.Context) {
					<-ctx.Done() // OnStop の cancel で解放される
					bodyCtxErr = ctx.Err()
					close(bodyCtxDone)
				},
				OnStopAux: func(context.Context) { auxStopped = true },
			}.Register(reg)

			require.NotNil(t, reg.start)
			require.NotNil(t, reg.stop)

			// OnStart: aux 起動 → Body goroutine 起動。
			require.NoError(t, reg.start(context.Background()))
			assert.True(t, auxStarted)

			// OnStop: cancel → Body 完了待ち → aux 停止。
			require.NoError(t, reg.stop(context.Background()))
			<-bodyCtxDone
			require.ErrorIs(t, bodyCtxErr, context.Canceled)
			assert.True(t, auxStopped)
		})

		t.Run("起動contextのキャンセルはBodyのcontextへ伝播しない", func(t *testing.T) {
			t.Parallel()

			bodyRan := make(chan struct{})
			var bodyCtxErr error

			reg := &fakeRegistrar{}
			SupervisedRunner{
				Body: func(ctx context.Context) {
					// Body 起動時点で起動 ctx のキャンセルに巻き込まれていないこと（Background 由来）。
					bodyCtxErr = ctx.Err()
					close(bodyRan)
				},
			}.Register(reg)

			// 既にキャンセル済みの起動 ctx を渡しても Body の ctx は無傷であること。
			startCtx, cancel := context.WithCancel(context.Background())
			cancel()
			require.NoError(t, reg.start(startCtx))

			<-bodyRan
			require.NoError(t, bodyCtxErr)

			// 後始末（goroutine リーク防止のため OnStop を呼ぶ）。
			require.NoError(t, reg.stop(context.Background()))
		})

		t.Run("Bodyが既に完了していればOnStopは即座に返る", func(t *testing.T) {
			t.Parallel()

			bodyDone := make(chan struct{})
			reg := &fakeRegistrar{}
			SupervisedRunner{
				Body: func(context.Context) { close(bodyDone) },
			}.Register(reg)

			require.NoError(t, reg.start(context.Background()))
			<-bodyDone // Body は即時完了

			require.NoError(t, reg.stop(context.Background()))
		})

		t.Run("Body完了前でも猶予切れ_stopCtx満了_でOnStopは返る", func(t *testing.T) {
			t.Parallel()

			block := make(chan struct{})
			reg := &fakeRegistrar{}
			SupervisedRunner{
				// ctx を無視してブロックし続ける Body（drain が完了しないケース）。
				Body: func(context.Context) { <-block },
			}.Register(reg)

			require.NoError(t, reg.start(context.Background()))

			// 既に満了した stopCtx を渡すと drain を待たず即座に返る。
			stopCtx, cancel := context.WithCancel(context.Background())
			cancel()
			require.NoError(t, reg.stop(stopCtx))

			close(block) // goroutine の後始末
		})

		t.Run("Bodyとauxがnilでも安全に登録_実行できる", func(t *testing.T) {
			t.Parallel()

			reg := &fakeRegistrar{}
			SupervisedRunner{}.Register(reg)

			require.NoError(t, reg.start(context.Background()))
			require.NoError(t, reg.stop(context.Background()))
		})
	})
}
