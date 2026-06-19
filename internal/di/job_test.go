package di

import (
	"context"
	"testing"
	"time"

	config "go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	gomock "go.uber.org/mock/gomock"
)

func TestNewJobCore(t *testing.T) {
	t.Parallel()

	// ジョブ用 fx グラフの結線が欠落なく成立することを検証する（コンストラクタの実体実行は伴わない）。
	// 本番と同じ fx.WithLogger(NewFxEventLogger) を渡し、ロガー構成子の依存解決も併せて検証する。
	require.NoError(t, fx.ValidateApp(NewJobCore(), fx.WithLogger(NewFxEventLogger)))
}

//nolint:paralleltest // EnsureRepoRootAndEnv が t.Setenv/t.Chdir を使用するため並列化不可
func TestNewJobCore_BootsWithMockedDB(t *testing.T) {
	// 実 DB を避けつつ、ジョブ用 fx グラフの全コンストラクタ実行とライフサイクル(OnStart/OnStop)を検証する。
	// DB ドライバを IF レベルでモックに差し替えて実 Ping を回避する（ジョブは HTTP サーバを起動しないためポート上書きは不要）。
	// EnsureRepoRootAndEnv が cwd を変更するため t.Parallel() は付けない。
	config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)

	ctrl := gomock.NewController(t)
	mockDB := mock_driver.NewMockDatabaseDriver(ctrl)
	mockDB.EXPECT().Close().Return(nil).AnyTimes()
	mockDB.EXPECT().Stats().Return(&pgxpool.Stat{}).AnyTimes()
	mockDB.EXPECT().Ping(gomock.Any()).Return(nil).AnyTimes()

	app := fx.New(
		NewJobCore(),
		fx.Replace(fx.Annotate(mockDB, fx.As(new(driver.DatabaseDriver)))),
		fx.NopLogger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, app.Start(ctx))

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()
	require.NoError(t, app.Stop(stopCtx))
}

//nolint:paralleltest // EnsureRepoRootAndEnv が t.Setenv/t.Chdir を使用するため並列化不可
func TestRunJob(t *testing.T) {
	config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)

	//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
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
		assert.Equal(t, context.Canceled, err)

		// チャンネルが閉じられていることを検証
		_, ok := <-done
		assert.False(t, ok)

		_ = stop(context.Background())
	})

	//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
	t.Run("stop: start していない状態で stop を呼ぶとエラーなしで成功することを期待する", func(t *testing.T) {
		_, stop := RunJob()
		require.NotNil(t, stop)

		err := stop(context.Background())
		require.NoError(t, err)
	})

	//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
	t.Run("stop: キャンセル済みコンテキストを与えると stop は context.Canceled を返すことを期待する", func(t *testing.T) {
		_, stop := RunJob()
		require.NotNil(t, stop)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := stop(ctx)
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("start: 存在しないジョブ名で start すると runner の unknown job エラーが done チャンネルに流れることを期待する", func(t *testing.T) {
		start, stop := RunJob()
		require.NotNil(t, start)
		require.NotNil(t, stop)

		// app.Start が完了できるように background コンテキストを使用
		done := start(context.Background(), "no-such-job", []string{})

		err := <-done
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown job")
		assert.Contains(t, err.Error(), "no-such-job")

		// チャンネルが閉じられていることを検証
		_, ok := <-done
		assert.False(t, ok)

		_ = stop(context.Background())
	})
}
