package hook

import (
	"context"
	"net"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/server"
	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	"go-boilerplate/internal/di/server/extension"
	mock_logging "go-boilerplate/internal/logging/mock"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRegisterHTTPServerHooks(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	var startFn func(context.Context) error
	var shutdownFn func(context.Context) error
	dummy := func(context.Context) error { return nil }

	mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
	mockReg.EXPECT().RegisterStart(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
		startFn = args[0].(func(context.Context) error)
	}).Times(1)

	mockLogger := mock_logging.NewMockLogger(ctrl)
	mockReg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
		shutdownFn = args[0].(func(context.Context) error)
	}).Times(1)

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	secCfg := config.NewSecurityConfig(cfg)
	srvCfg := config.NewServerConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	e := server.NewAppServer(srvCfg)

	RegisterHTTPServerHooks(e, mockReg, mockLogger, appCfg, secCfg, srvCfg, osCfg, &extension.AppliedServerExtends{})
	require.NotNil(t, startFn)
	require.NotNil(t, shutdownFn)
}

func Test_newStartServerFunc(t *testing.T) {
	t.Parallel()

	t.Run("HTTPサーバーが起動後にListenerが設定されること", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockLogger := mock_logging.NewMockLogger(ctrl)
		namedMock := mock_logging.NewMockLogger(ctrl)

		mockLogger.EXPECT().Named("server.Start").Return(namedMock).AnyTimes()
		namedMock.EXPECT().CallerSkip(serverCallerSkip).Return(namedMock).AnyTimes()
		namedMock.EXPECT().
			Info("http started", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)
		namedMock.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()

		lc := &net.ListenConfig{}
		ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)
		port := ln.Addr().(*net.TCPAddr).Port
		require.NoError(t, err)
		require.NoError(t, ln.Close())

		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		secCfg := config.NewSecurityConfig(cfg)
		srvCfg := config.NewServerConfig(cfg)
		osCfg := config.NewOperationSystemConfig(cfg)
		srvCfg.SetServerPort(t, port)

		e := server.NewAppServer(srvCfg)

		fn := newStartServerFunc(e, srvCfg, mockLogger, secCfg, appCfg, osCfg)
		err = fn(context.Background())
		require.NoError(t, err)
		require.NotNil(t, e.Listener)

		t.Cleanup(func() {
			_ = e.Shutdown(context.Background())
		})
	})

	t.Run("ポートのリッスンに失敗した場合、エラーが返されること", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockLogger := mock_logging.NewMockLogger(ctrl)

		lc := &net.ListenConfig{}
		ln, err := lc.Listen(context.Background(), "tcp", ":0")
		require.NoError(t, err)
		port := ln.Addr().(*net.TCPAddr).Port
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = ln.Close()
		})

		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		secCfg := config.NewSecurityConfig(cfg)
		srvCfg := config.NewServerConfig(cfg)
		osCfg := config.NewOperationSystemConfig(cfg)
		srvCfg.SetServerPort(t, port)

		e := server.NewAppServer(srvCfg)
		fn := newStartServerFunc(e, srvCfg, mockLogger, secCfg, appCfg, osCfg)

		err = fn(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to listen on port")
	})
}

func Test_newStopServerFunc(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	mockLogger := mock_logging.NewMockLogger(ctrl)

	mockLogger.EXPECT().Named("server.Stop").Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().CallerSkip(serverCallerSkip).Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().Info("http stopping", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	cfg := config.MockConfigForTest(t)
	srvCfg := config.NewServerConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	e := server.NewAppServer(srvCfg)
	fn := newStopServerFunc(e, mockLogger, osCfg)

	require.NoError(t, fn(context.Background()))
}
