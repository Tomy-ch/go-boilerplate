package worker

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/xerrors"
)

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
				return done // 自走停止しない（SIGTERM 待ち）
			}
			stop := func(context.Context) error {
				stopped.Store(true)
				done <- nil // drain 完了で engine が結果を書く
				return nil
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // SIGTERM 相当

			err := runWorker(ctx, "w", nil, start, stop)

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

			err := runWorker(context.Background(), "w", nil, start, stop)

			require.ErrorIs(t, err, wantErr)
			assert.True(t, stopped.Load())
		})

		t.Run("SIGTERM と drain 中の Fatal が競合しても Fatal を取りこぼさない", func(t *testing.T) {
			t.Parallel()

			// drain 中に engine が Fatal を検出して done へ書くケース。
			// 非ブロッキング再確認だと取りこぼして nil(exit 0) になってしまうため、必ず待ち切ることを検証する。
			wantErr := xerrors.New("fatal during drain")
			done := make(chan error, 1)
			start := func(context.Context, string, []string) <-chan error { return done }
			stop := func(context.Context) error {
				done <- wantErr // drain 完了後に Fatal を報告
				return nil
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // SIGTERM 相当

			err := runWorker(ctx, "w", nil, start, stop)

			require.ErrorIs(t, err, wantErr)
		})
	})
}

func Test_gracefulStop(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("stop に有効期限付き context を渡して呼び出す", func(t *testing.T) {
			t.Parallel()

			var gotDeadline atomic.Bool
			stop := func(ctx context.Context) error {
				_, ok := ctx.Deadline()
				gotDeadline.Store(ok)
				return nil
			}

			gracefulStop(context.Background(), stop)

			assert.True(t, gotDeadline.Load())
			// stopTimeout が将来 0 にされないことの軽い保証。
			assert.Positive(t, int64(stopTimeout))
		})
	})
}
