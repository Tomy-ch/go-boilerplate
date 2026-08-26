package di

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	gomock "go.uber.org/mock/gomock"

	config "go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
)

func TestNewWorkerCore(t *testing.T) {
	t.Parallel()

	require.NoError(t, fx.ValidateApp(NewWorkerCore(), fx.WithLogger(NewFxEventLogger)))
}

func TestNewWorkerCore_BootsWithMockedDB(t *testing.T) {
	// worker 未選択(state 未設定)でも起動・停止でき、health listener の起動/停止まで通ることを確認する。
	config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)
	// health listener のポート衝突を避けるため OS 割り当ての空きポートを使う。
	t.Setenv("WORKER_HEALTH_LISTEN_ADDR", "127.0.0.1:0")

	ctrl := gomock.NewController(t)
	mockDB := mock_driver.NewMockDatabaseDriver(ctrl)
	mockDB.EXPECT().Close().Return(nil).AnyTimes()
	mockDB.EXPECT().Stats().Return(&pgxpool.Stat{}).AnyTimes()
	mockDB.EXPECT().Ping(gomock.Any()).Return(nil).AnyTimes()

	app := fx.New(
		NewWorkerCore(),
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
func TestRunWorker(t *testing.T) {
	config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)

	//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
		t.Run("stop: start していない状態で stop を呼ぶとエラーなしで成功する", func(t *testing.T) {
			_, stop := RunWorker(30 * time.Second)
			require.NotNil(t, stop)

			require.NoError(t, stop(context.Background()))
		})

		//nolint:paralleltest // t.Setenv を使用するため並列化不可
		t.Run("start: 正常に起動すると done を返し stop で正常終了する", func(t *testing.T) {
			// health listener のポート衝突を避けるため OS 割り当ての空きポートを使う。
			t.Setenv("WORKER_HEALTH_LISTEN_ADDR", "127.0.0.1:0")

			start, stop := RunWorker(30 * time.Second)
			require.NotNil(t, start)
			require.NotNil(t, stop)

			done := start(context.Background(), "no-worker", nil)
			require.NotNil(t, done)
			// no-worker は未登録のため engine が即座にエラーを done へ返す（起動自体は成功）。
			require.Error(t, <-done)

			require.NoError(t, stop(context.Background()))
		})
	})

	//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
	t.Run("異常系", func(t *testing.T) {
		//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
		t.Run("start: キャンセル済みコンテキストで開始すると context.Canceled を返してチャンネルを閉じる", func(t *testing.T) {
			start, stop := RunWorker(30 * time.Second)
			require.NotNil(t, start)
			require.NotNil(t, stop)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			done := start(ctx, "no-worker", nil)
			err := <-done
			require.ErrorIs(t, err, context.Canceled)

			_, ok := <-done
			require.False(t, ok)

			require.NoError(t, stop(context.Background()))
		})

		//nolint:paralleltest // t.Setenv を使用するため並列化不可
		t.Run("stop: 起動後に期限切れ context で停止するとエラーを返す", func(t *testing.T) {
			t.Setenv("WORKER_HEALTH_LISTEN_ADDR", "127.0.0.1:0")

			start, stop := RunWorker(30 * time.Second)
			require.NotNil(t, start)
			require.NotNil(t, stop)

			done := start(context.Background(), "no-worker", nil)
			require.Error(t, <-done)

			// 既にキャンセル済みの context で停止すると app.Stop がエラーを返す。
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			require.Error(t, stop(ctx))
		})

		//nolint:paralleltest // t.Setenv を使用するため並列化不可
		t.Run("start: fx グラフの構築に失敗すると panic せず構築エラーを閉じ済みチャンネルで返す", func(t *testing.T) {
			t.Setenv("APP_SHUTDOWN_TIMEOUT", "not-a-duration")

			start, stop := RunWorker(30 * time.Second)
			require.NotNil(t, start)
			require.NotNil(t, stop)

			// nil 参照の退行が起きても panic をこのテスト内で捕捉し、同一パッケージの後続テストを巻き込まない。
			var done <-chan error
			require.NotPanics(t, func() {
				done = start(context.Background(), "no-worker", nil)
			})

			require.ErrorIs(t, <-done, config.ErrFailedToParseConfig)

			_, ok := <-done
			require.False(t, ok)
		})

		//nolint:paralleltest // t.Setenv を使用するため並列化不可
		t.Run("stop: fx グラフの構築に失敗すると panic せず構築エラーを返す", func(t *testing.T) {
			t.Setenv("APP_SHUTDOWN_TIMEOUT", "not-a-duration")

			_, stop := RunWorker(30 * time.Second)
			require.NotNil(t, stop)

			// nil 参照の退行が起きても panic をこのテスト内で捕捉し、同一パッケージの後続テストを巻き込まない。
			var err error
			require.NotPanics(t, func() {
				err = stop(context.Background())
			})

			require.ErrorIs(t, err, config.ErrFailedToParseConfig)
		})
	})
}
