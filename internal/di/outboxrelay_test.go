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
	t.Parallel()

	require.NotNil(t, NewOutboxRelayApp())
}

//nolint:paralleltest // EnsureRepoRootAndEnv が t.Setenv/t.Chdir を使用するため並列化不可
func TestRunOutboxReplay(t *testing.T) {
	config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// dead 行が無い状態でも 0 件・エラーなしで完了する（ワンショット実行の起動・停止まで通す）。
	count, err := RunOutboxReplay(ctx, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}
