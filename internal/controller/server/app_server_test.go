package server

import (
	"context"
	"testing"

	"boilerplate-go/internal/config"
	mock_lifecycle "boilerplate-go/internal/di/lifecycle/mock"
	"boilerplate-go/internal/di/server/extension"
	mock_logging "boilerplate-go/internal/logging/mock"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestNewAppServer(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	srvCfg := config.NewServerConfig(cfg)

	actual := NewAppServer(srvCfg)
	require.Equal(t, srvCfg.ReadHeaderTimeout(), actual.Server.ReadHeaderTimeout)
	require.Equal(t, srvCfg.ReadTimeout(), actual.Server.ReadTimeout)
	require.Equal(t, srvCfg.WriteTimeout(), actual.Server.WriteTimeout)
	require.Equal(t, srvCfg.IdleTimeout(), actual.Server.IdleTimeout)
	require.NotNil(t, actual)
}

func TestServeHTTP(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	e := NewAppServer(srvCfg)

	ServeHTTP(e, mockReg, mockLogger, appCfg, secCfg, srvCfg, &extension.AppliedServerExtends{})

	require.NotNil(t, startFn)
	require.NotNil(t, shutdownFn)
}

func Test_newStartServerFunc(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mock_logging.NewMockLogger(ctrl)
	namedMock := mock_logging.NewMockLogger(ctrl)

	mockLogger.EXPECT().Named("server.Start").Return(namedMock).AnyTimes()
	namedMock.EXPECT().Info("http started", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	namedMock.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	secCfg := config.NewSecurityConfig(cfg)
	srvCfg := config.NewServerConfig(cfg)

	e := NewAppServer(srvCfg)

	fn := newStartServerFunc(e, srvCfg, mockLogger, secCfg, appCfg)

	require.NoError(t, fn(context.Background()))
}

func Test_newStopServerFunc(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mock_logging.NewMockLogger(ctrl)

	mockLogger.EXPECT().Named("server.Stop").Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().Info("http stopping").Times(1)

	cfg := config.MockConfigForTest(t)
	srvCfg := config.NewServerConfig(cfg)

	e := NewAppServer(srvCfg)
	fn := newStopServerFunc(e, mockLogger)

	require.NoError(t, fn(context.Background()))
}
