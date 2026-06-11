package hook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/config"

	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	mock_shutdowner "go-boilerplate/internal/di/shutdowner/mock"
	mock_logging "go-boilerplate/internal/logging/mock"
	mock_job "go-boilerplate/internal/usecase/boundary/job/mock"
)

func TestRegisterJobHooks(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("startフックが登録されgoroutineがジョブを実行する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			var startFn func(context.Context) error
			reg := mock_lifecycle.NewMockRegistrar(ctrl)
			sd := mock_shutdowner.NewMockShutdowner(ctrl)
			runner := mock_job.NewMockRunner(ctrl)
			logger := mock_logging.NewMockLogger(ctrl)

			dummy := func(context.Context) error { return nil }
			reg.EXPECT().RegisterStart(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
				startFn = args[0].(func(context.Context) error)
			}).Times(1)

			doneCh := make(chan error, 1)
			state := mock_job.NewMockState(ctrl)
			state.EXPECT().Snapshot().Return("job-x", []string{"a"}, doneCh).Times(1)
			runner.EXPECT().Run(gomock.Any(), "job-x", []string{"a"}).Return(nil).Times(1)
			sd.EXPECT().Shutdown().Return(nil).Times(1)

			osCfg := config.NewOperatingSystemConfig(config.MockConfigForTest(t))
			RegisterJobHooks(reg, sd, runner, logger, osCfg, state)

			require.NotNil(t, startFn)
			require.NoError(t, startFn(context.Background()))

			// goroutine の完了を done のクローズで待つ（sleep 不要・決定的）。
			require.NoError(t, <-doneCh)
			_, ok := <-doneCh
			require.False(t, ok)
		})
	})
}

func TestRunJobAndShutdown(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ジョブが無い場合はログ出力のみで停止する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sd := mock_shutdowner.NewMockShutdowner(ctrl)
			runner := mock_job.NewMockRunner(ctrl)
			logger := mock_logging.NewMockLogger(ctrl)
			named := mock_logging.NewMockLogger(ctrl)

			state := mock_job.NewMockState(ctrl)
			state.EXPECT().Snapshot().Return("", []string{}, nil).Times(1)
			logger.EXPECT().Named("job.Hooks").Return(named).Times(1)
			named.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
			sd.EXPECT().Shutdown().Return(nil).Times(1)
			runner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			osCfg := config.NewOperatingSystemConfig(config.MockConfigForTest(t))
			runJobAndShutdown(context.Background(), sd, runner, logger, osCfg, state)
		})

		t.Run("ジョブがある場合はrunnerが実行されdoneに結果が送られる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sd := mock_shutdowner.NewMockShutdowner(ctrl)
			runner := mock_job.NewMockRunner(ctrl)
			logger := mock_logging.NewMockLogger(ctrl)

			doneCh := make(chan error, 1)
			state := mock_job.NewMockState(ctrl)
			state.EXPECT().Snapshot().Return("job-x", []string{"a", "b"}, doneCh).Times(1)
			runner.EXPECT().Run(gomock.Any(), "job-x", []string{"a", "b"}).Return(nil).Times(1)
			sd.EXPECT().Shutdown().Return(nil).Times(1)

			osCfg := config.NewOperatingSystemConfig(config.MockConfigForTest(t))
			// 直接同期呼び出し。戻った時点で done に結果が入りクローズ済み。
			runJobAndShutdown(context.Background(), sd, runner, logger, osCfg, state)

			require.NoError(t, <-doneCh)
			_, ok := <-doneCh
			require.False(t, ok)
		})

		t.Run("起動contextがキャンセル済みでもジョブのcontextは中断されず実行される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sd := mock_shutdowner.NewMockShutdowner(ctrl)
			runner := mock_job.NewMockRunner(ctrl)
			logger := mock_logging.NewMockLogger(ctrl)

			doneCh := make(chan error, 1)
			state := mock_job.NewMockState(ctrl)
			state.EXPECT().Snapshot().Return("job-x", []string{}, doneCh).Times(1)

			// runner が受け取る context が起動 ctx のキャンセルに巻き込まれていないこと（Err() == nil）を確認する。
			// context.WithoutCancel が無い回帰が起きると、ここで context.Canceled になる。
			var jobCtxErr error
			runner.EXPECT().Run(gomock.Any(), "job-x", gomock.Any()).DoAndReturn(
				func(ctx context.Context, _ string, _ []string) error {
					jobCtxErr = ctx.Err()
					return nil
				}).Times(1)
			sd.EXPECT().Shutdown().Return(nil).Times(1)

			osCfg := config.NewOperatingSystemConfig(config.MockConfigForTest(t))

			startCtx, cancel := context.WithCancel(context.Background())
			cancel()
			runJobAndShutdown(startCtx, sd, runner, logger, osCfg, state)

			require.NoError(t, <-doneCh)
			assert.NoError(t, jobCtxErr)
		})
	})
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Shutdown失敗時はエラーログが出力される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sd := mock_shutdowner.NewMockShutdowner(ctrl)
			logger := mock_logging.NewMockLogger(ctrl)
			named := mock_logging.NewMockLogger(ctrl)

			sd.EXPECT().Shutdown().Return(assert.AnError).Times(1)
			logger.EXPECT().Named("job.Hooks").Return(named).Times(1)
			named.EXPECT().Error(gomock.Any(), gomock.Any()).Times(1)

			shutdown(sd, logger)
		})
	})
}
