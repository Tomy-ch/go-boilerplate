package hook

import (
	"context"
	"testing"
	"time"

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

	t.Run("ジョブが無い場合はShutdownされログが出力される", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		var startFn func(context.Context) error

		reg := mock_lifecycle.NewMockRegistrar(ctrl)
		sd := mock_shutdowner.NewMockShutdowner(ctrl)
		runner := mock_job.NewMockRunner(ctrl)
		logger := mock_logging.NewMockLogger(ctrl)
		named := mock_logging.NewMockLogger(ctrl)

		// 登録された start 関数を取得する
		dummy := func(context.Context) error { return nil }
		reg.EXPECT().RegisterStart(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
			startFn = args[0].(func(context.Context) error)
		}).Times(1)

		// Snapshot が nil を返す場合：ログ出力と Shutdown を行う経路
		state := mock_job.NewMockState(ctrl)
		state.EXPECT().Snapshot().Return("", []string{}, nil).Times(1)

		// ロガーの Named と Info の呼び出しを期待する
		logger.EXPECT().Named("job.Hooks").Return(named).Times(1)
		named.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

		sd.EXPECT().Shutdown().Return(nil).Times(1)

		// runner は呼ばれないことを期待する
		runner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		cfg := config.MockConfigForTest(t)
		osCfg := config.NewOperationSystemConfig(cfg)

		RegisterJobHooks(reg, sd, runner, logger, osCfg, state)

		// 登録された start フックを呼び出す
		require.NotNil(t, startFn)
		require.NoError(t, startFn(context.Background()))

		// ゴルーチンが実行される時間を与える（Shutdown 呼び出しは ctrl.Finish で gomock が検証）
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("ジョブがある場合はrunnerが実行されdoneに結果が送られる", func(t *testing.T) {
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

		// バッファ付きの done チャネルを持つ state を準備
		doneCh := make(chan error, 1)
		state := mock_job.NewMockState(ctrl)
		state.EXPECT().Snapshot().Return("job-x", []string{"a", "b"}, doneCh).Times(1)
		runner.EXPECT().Run(gomock.Any(), "job-x", []string{"a", "b"}).DoAndReturn(func(_ context.Context, _ string, _ []string) error {
			return nil
		}).Times(1)

		sd.EXPECT().Shutdown().Return(nil).Times(1)

		cfg := config.MockConfigForTest(t)
		osCfg := config.NewOperationSystemConfig(cfg)

		RegisterJobHooks(reg, sd, runner, logger, osCfg, state)

		require.NotNil(t, startFn)
		require.NoError(t, startFn(context.Background()))

		// done チャネルから結果を待つ（runner は nil を返す想定）
		select {
		case v := <-doneCh:
			require.NoError(t, v)
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("did not receive done value")
		}

		// ゴルーチンが実行される時間を与える（Shutdown 呼び出しは ctrl.Finish で gomock が検証）
		time.Sleep(50 * time.Millisecond)
	})
}
