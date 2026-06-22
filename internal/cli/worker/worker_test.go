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

			var stopped atomic.Bool
			start := func(context.Context, string, []string) <-chan error {
				return make(chan error) // 自走停止しない（SIGTERM 待ち）
			}
			stop := func(context.Context) error {
				stopped.Store(true)
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

			gracefulStop(stop)

			assert.True(t, gotDeadline.Load())
			// stopTimeout が将来 0 にされないことの軽い保証。
			assert.Positive(t, int64(stopTimeout))
		})
	})
}
