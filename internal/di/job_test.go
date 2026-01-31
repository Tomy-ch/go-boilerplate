package di

import (
	"context"
	"testing"

	config "boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestNewJobCore(t *testing.T) {
	t.Parallel()

	jobApp := NewJobCore()
	require.NotNil(t, jobApp)
}

func TestRunJob(t *testing.T) {
	config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)

	t.Run("start: キャンセル済みコンテキストで開始すると start は context.Canceled を返してチャンネルを閉じることを期待する", func(t *testing.T) {
		start, stop := RunJob()
		require.NotNil(t, start)
		require.NotNil(t, stop)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := start(ctx, "no-job", []string{})
		// start は err を送ってチャンネルを閉じる
		err := <-done
		require.Error(t, err)
		require.Equal(t, context.Canceled, err)

		// チャンネルが閉じられていることを検証
		_, ok := <-done
		require.False(t, ok)

		_ = stop(context.Background())
	})

	t.Run("stop: start していない状態で stop を呼ぶとエラーなしで成功することを期待する", func(t *testing.T) {
		_, stop := RunJob()
		require.NotNil(t, stop)

		err := stop(context.Background())
		require.NoError(t, err)
	})

	t.Run("stop: キャンセル済みコンテキストを与えると stop は context.Canceled を返すことを期待する", func(t *testing.T) {
		_, stop := RunJob()
		require.NotNil(t, stop)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := stop(ctx)
		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
	})

	t.Run("start: 存在しないジョブ名で start すると runner の unknown job エラーが done チャンネルに流れることを期待する", func(t *testing.T) {
		start, stop := RunJob()
		require.NotNil(t, start)
		require.NotNil(t, stop)

		// app.Start が完了できるように background コンテキストを使用
		done := start(context.Background(), "no-such-job", []string{})

		err := <-done
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown job: no-such-job")

		// チャンネルが閉じられていることを検証
		_, ok := <-done
		require.False(t, ok)

		_ = stop(context.Background())
	})
}
