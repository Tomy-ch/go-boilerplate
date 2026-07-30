package di

import (
	"context"
	"errors"
	"testing"
	"time"

	config "go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	"go-boilerplate/internal/logging"

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
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
		t.Run("stop: start していない状態で stop を呼ぶとエラーなしで成功することを期待する", func(t *testing.T) {
			_, stop := RunJob(30 * time.Second)
			require.NotNil(t, stop)

			err := stop(context.Background())
			require.NoError(t, err)
		})
	})

	//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
	t.Run("異常系", func(t *testing.T) {
		//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
		t.Run("start: キャンセル済みコンテキストで開始すると start は context.Canceled を返してチャンネルを閉じることを期待する", func(t *testing.T) {
			start, stop := RunJob(30 * time.Second)
			require.NotNil(t, start)
			require.NotNil(t, stop)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			done := start(ctx, "no-job", []string{})
			// start は err を送ってチャンネルを閉じる
			err := <-done
			require.Error(t, err)
			require.ErrorIs(t, err, context.Canceled)

			// チャンネルが閉じられていることを検証
			_, ok := <-done
			assert.False(t, ok)

			_ = stop(context.Background())
		})

		//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
		t.Run("stop: キャンセル済みコンテキストを与えると stop は context.Canceled を返すことを期待する", func(t *testing.T) {
			_, stop := RunJob(30 * time.Second)
			require.NotNil(t, stop)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := stop(ctx)
			require.Error(t, err)
			require.ErrorIs(t, err, context.Canceled)
		})

		//nolint:paralleltest // 親が EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
		t.Run("start: 存在しないジョブ名で start すると runner の unknown job エラーが done チャンネルに流れることを期待する", func(t *testing.T) {
			start, stop := RunJob(30 * time.Second)
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

		//nolint:paralleltest // t.Setenv を使用するため並列化不可
		t.Run("start: fx グラフの構築に失敗すると panic せず構築エラーを閉じ済みチャンネルで返すことを期待する", func(t *testing.T) {
			t.Setenv("APP_SHUTDOWN_TIMEOUT", "not-a-duration")

			start, stop := RunJob(30 * time.Second)
			require.NotNil(t, start)
			require.NotNil(t, stop)

			// nil 参照の退行が起きても panic をこのテスト内で捕捉し、同一パッケージの後続テストを巻き込まない。
			var done <-chan error
			require.NotPanics(t, func() {
				done = start(context.Background(), "no-job", []string{})
			})

			require.ErrorIs(t, <-done, config.ErrFailedToParseConfig)

			_, ok := <-done
			assert.False(t, ok)
		})

		//nolint:paralleltest // t.Setenv を使用するため並列化不可
		t.Run("stop: fx グラフの構築に失敗すると panic せず構築エラーを返すことを期待する", func(t *testing.T) {
			t.Setenv("APP_SHUTDOWN_TIMEOUT", "not-a-duration")

			_, stop := RunJob(30 * time.Second)
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

func Test_failClosedChan(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡したエラーを 1 件送信した後クローズ済みのチャンネルを返す", func(t *testing.T) {
			t.Parallel()

			wantErr := errors.New("build failed")

			ch := failClosedChan(wantErr)
			require.ErrorIs(t, <-ch, wantErr)

			_, ok := <-ch
			assert.False(t, ok)
		})
	})
}

func Test_jobEventFields(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("イベント種別とタイムゾーンを引数の値で付与する", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)
			logger.Info(context.Background(), "job event",
				jobEventFields(logging.EventTypeStart, "Asia/Tokyo")...)

			require.Equal(t, 1, logs.Len())
			fields := logs.All()[0].ContextMap()
			assert.Equal(t, logging.EventTypeStart, fields[logging.EventTypeKey])
			assert.Equal(t, "Asia/Tokyo", fields[logging.EventTzKey])
		})

		t.Run("発生時刻に呼び出し時点の現在時刻を付与する", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)

			before := time.Now()
			fields := jobEventFields(logging.EventTypeEnd, "UTC")
			after := time.Now()
			logger.Info(context.Background(), "job event", fields...)

			require.Equal(t, 1, logs.Len())
			rawAt, ok := logs.All()[0].ContextMap()[logging.EventAtKey].(string)
			require.True(t, ok)

			eventAt, err := time.Parse(time.RFC3339Nano, rawAt)
			require.NoError(t, err)
			assert.False(t, eventAt.Before(before))
			assert.False(t, eventAt.After(after))
		})
	})
}
