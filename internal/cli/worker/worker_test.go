package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/xerrors"
)

// testGrace は、テスト用の停止猶予（APP_SHUTDOWN_TIMEOUT 相当）です。
const testGrace = 30 * time.Second

func Test_runWorker(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SIGTERM(ctx 完了)で停止し nil を返す", func(t *testing.T) {
			t.Parallel()

			// engine は drain 完了時に done へ結果（clean stop=nil）を書く。
			done := make(chan error, 1)
			var stopped atomic.Bool
			start := func(context.Context, string, []string) <-chan error {
				return done
			}
			stop := func(context.Context) error {
				stopped.Store(true)
				done <- nil // drain 完了で engine が結果を書く
				return nil
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // SIGTERM 相当

			err := runWorker(ctx, "w", nil, testGrace, start, stop)

			require.NoError(t, err)
			assert.True(t, stopped.Load())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("engine 自走停止時は done のエラーを返し stop も呼ぶ", func(t *testing.T) {
			t.Parallel()

			wantErr := xerrors.New("fatal")
			done := make(chan error, 1)
			done <- wantErr
			var stopped atomic.Bool
			start := func(context.Context, string, []string) <-chan error { return done }
			stop := func(context.Context) error {
				stopped.Store(true)
				return nil
			}

			err := runWorker(context.Background(), "w", nil, testGrace, start, stop)

			require.ErrorIs(t, err, wantErr)
			assert.True(t, stopped.Load())
		})

		t.Run("SIGTERM と drain 中の Fatal が競合しても Fatal を取りこぼさない", func(t *testing.T) {
			t.Parallel()

			wantErr := xerrors.New("fatal during drain")
			done := make(chan error, 1)
			start := func(context.Context, string, []string) <-chan error { return done }
			stop := func(context.Context) error {
				done <- wantErr // drain 完了後に Fatal を報告
				return nil
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // SIGTERM 相当

			err := runWorker(ctx, "w", nil, testGrace, start, stop)

			require.ErrorIs(t, err, wantErr)
		})
	})
}

func Test_gracefulStop(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("stop に grace 由来の有効期限付き context を渡して呼び出す", func(t *testing.T) {
			t.Parallel()

			var (
				gotDeadline atomic.Bool
				remaining   atomic.Int64
			)
			stop := func(ctx context.Context) error {
				dl, ok := ctx.Deadline()
				gotDeadline.Store(ok)
				if ok {
					remaining.Store(int64(time.Until(dl)))
				}
				return nil
			}

			gracefulStop(context.Background(), testGrace, stop)

			require.True(t, gotDeadline.Load())
			// 停止猶予は grace 由来。ライブ時刻に依存しないよう grace/2 超を下限として固定する。
			assert.Greater(t, remaining.Load(), int64(testGrace/2))
		})

		t.Run("親ctxがキャンセル済みでも停止用ctxは期限切れでない", func(t *testing.T) {
			t.Parallel()

			// gracefulStop は stop を同期呼び出しするため平易な変数で記録できる。
			var (
				called      bool
				hasDeadline bool
				ctxErr      error
			)
			stop := func(ctx context.Context) error {
				called = true
				_, hasDeadline = ctx.Deadline()
				ctxErr = ctx.Err()
				return nil
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // SIGTERM 相当に親をキャンセル済みにする

			gracefulStop(ctx, testGrace, stop)

			require.True(t, called, "停止処理が呼ばれること")
			require.True(t, hasDeadline, "停止用 context に deadline があること")
			// context.WithoutCancel により親のキャンセルは引き継がれず、grace 由来の猶予が確保される。
			require.NoError(t, ctxErr, "停止用 context が期限切れでないこと")
		})
	})
}

func TestRunWorkerWith(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("provideで取得したstart/stopをrunWorkerへ渡し結果を返す", func(t *testing.T) {
			t.Parallel()

			done := make(chan error, 1)
			var stopped atomic.Bool
			start := func(context.Context, string, []string) <-chan error { return done }
			stop := func(context.Context) error { //nolint:unparam // StopFunc シグネチャ準拠のため error を返す
				stopped.Store(true)
				done <- nil // drain 完了で engine が結果を書く
				return nil
			}
			provide := func() (StartFunc, StopFunc) { return start, stop }

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // SIGTERM 相当

			err := RunWorkerWith(ctx, "w", nil, testGrace, provide)

			require.NoError(t, err)
			assert.True(t, stopped.Load(), "provide 由来の停止処理が呼ばれること")
		})
	})
}
