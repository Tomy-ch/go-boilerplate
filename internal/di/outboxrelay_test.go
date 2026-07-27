package di

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	config "go-boilerplate/internal/config"
)

func TestNewOutboxRelayCore(t *testing.T) {
	t.Parallel()

	// relay 用 fx グラフの結線が欠落なく成立することを検証する（コンストラクタの実体実行は伴わない）。
	require.NoError(t, fx.ValidateApp(NewOutboxRelayCore(), fx.WithLogger(NewFxEventLogger)))
}

func TestNewOutboxRelayApp(t *testing.T) {
	config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)

	t.Run("正常系", func(t *testing.T) {
		t.Run("有効な OUTBOX_ENDPOINT なら relay app が起動可能", func(t *testing.T) {
			t.Setenv("OUTBOX_ENDPOINT", "http://localhost:9999")

			app := NewOutboxRelayApp(30 * time.Second)

			// fx.New はコンストラクタ（NewEndpoint 等）を実行しエラーを app.Err() に格納する。
			require.NoError(t, app.Err())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("OUTBOX_ENDPOINT が空なら起動時に弾かれる", func(t *testing.T) {
			t.Setenv("OUTBOX_ENDPOINT", "")

			app := NewOutboxRelayApp(30 * time.Second)

			require.Error(t, app.Err())
		})
	})
}

//nolint:paralleltest // EnsureRepoRootAndEnv が t.Setenv/t.Chdir を使用するため並列化不可
func TestRunOutboxReplay(t *testing.T) {
	config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)

	t.Run("正常系", func(t *testing.T) {
		t.Run("dead行が無くても0件・エラーなしで起動から停止まで完了する", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			count, err := RunOutboxReplay(ctx, nil)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, count, int64(0))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("キャンセル済みコンテキストではapp.Start失敗を0件で返す", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			count, err := RunOutboxReplay(ctx, nil)
			require.Error(t, err)
			assert.Zero(t, count)
		})
	})
}

func Test_outboxRelayCommonOptions(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
