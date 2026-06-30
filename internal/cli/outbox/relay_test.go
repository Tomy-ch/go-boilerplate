package outbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	outboxcli "go-boilerplate/internal/cli/outbox"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRelay(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("起動後 ctx 完了でグレースフルに停止する", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // 即時に <-ctx.Done() を抜けさせる

			started, stopped := false, false
			start := func(context.Context) error { started = true; return nil }
			stop := func(stopCtx context.Context) error {
				stopped = true
				// 停止用 context には shutdownTimeout 由来の deadline が設定されている。
				_, ok := stopCtx.Deadline()
				assert.True(t, ok)
				return nil
			}

			require.NoError(t, outboxcli.RunRelay(ctx, time.Second, start, stop))
			assert.True(t, started)
			assert.True(t, stopped)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("起動失敗時は停止せずエラーを返す", func(t *testing.T) {
			t.Parallel()

			wantErr := errors.New("start failed")
			stopped := false
			start := func(context.Context) error { return wantErr }
			stop := func(context.Context) error { stopped = true; return nil }

			err := outboxcli.RunRelay(context.Background(), time.Second, start, stop)
			require.ErrorIs(t, err, wantErr)
			assert.False(t, stopped)
		})

		t.Run("停止失敗時は起動済みでも停止エラーを返す", func(t *testing.T) {
			t.Parallel()

			wantErr := errors.New("stop failed")
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // 即時に <-ctx.Done() を抜けさせる

			started := false
			start := func(context.Context) error { started = true; return nil }
			stop := func(context.Context) error { return wantErr }

			err := outboxcli.RunRelay(ctx, time.Second, start, stop)
			require.ErrorIs(t, err, wantErr)
			assert.True(t, started)
		})
	})
}
