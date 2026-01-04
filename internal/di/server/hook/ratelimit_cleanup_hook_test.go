package hook

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"boilerplate-go/internal/config"
	mock_ratelimit "boilerplate-go/internal/controller/httpstack/ratelimit/mock"
	mock_lifecycle "boilerplate-go/internal/di/lifecycle/mock"
	mock_logging "boilerplate-go/internal/logging/mock"
)

func TestRegisterRateLimitHooks(t *testing.T) {
	t.Parallel()

	t.Run("IPRateLimitが無効な場合は登録せず終了する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		reg := mock_lifecycle.NewMockRegistrar(ctrl)
		rl := mock_ratelimit.NewMockIPRateLimiter(ctrl)
		mockLogger := mock_logging.NewMockLogger(ctrl)

		mockLogger.EXPECT().Named("ratelimit.Hooks").Return(mockLogger).AnyTimes()
		mockLogger.EXPECT().CallerSkip(ratelimitCallerSkip).Return(mockLogger).AnyTimes()
		mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		mockLogger.EXPECT().
			Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes()
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()

		ipCfg := &config.IPRateLimitConfig{}
		osCfg := config.NewOperationSystemConfig(config.MockConfigForTest(t))

		RegisterRateLimitHooks(reg, rl, mockLogger, osCfg, ipCfg)
	})

	t.Run("IPRateLimitが有効な場合はフック登録される", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		reg := mock_lifecycle.NewMockRegistrar(ctrl)
		rl := mock_ratelimit.NewMockIPRateLimiter(ctrl)
		mockLogger := mock_logging.NewMockLogger(ctrl)

		mockLogger.EXPECT().Named("ratelimit.Hooks").Return(mockLogger).AnyTimes()
		mockLogger.EXPECT().CallerSkip(ratelimitCallerSkip).Return(mockLogger).AnyTimes()
		mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		mockLogger.EXPECT().
			Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes()
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()

		cfg := config.MockConfigForTest(t)
		ipCfg := config.NewIPRateLimitConfig(cfg)
		osCfg := config.NewOperationSystemConfig(cfg)

		reg.EXPECT().RegisterStart(gomock.Any())
		reg.EXPECT().RegisterStop(gomock.Any())

		RegisterRateLimitHooks(reg, rl, mockLogger, osCfg, ipCfg)
	})
}

func Test_newRateLimitCleanupHook(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	rl := mock_ratelimit.NewMockIPRateLimiter(ctrl)
	mockLogger := mock_logging.NewMockLogger(ctrl)

	cfg := config.MockConfigForTest(t)
	osCfg := config.NewOperationSystemConfig(cfg)
	ipCfg := config.NewIPRateLimitConfig(cfg)

	expected := &rateLimitCleanupHook{
		rl:     rl,
		logger: mockLogger,
		osCfg:  osCfg,
		ipCfg:  ipCfg,
	}

	actual := newRateLimitCleanupHook(rl, mockLogger, osCfg, ipCfg)
	require.Equal(t, expected, actual)
}

func Test_rateLimitCleanupHook_Register(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	reg := mock_lifecycle.NewMockRegistrar(ctrl)
	rl := mock_ratelimit.NewMockIPRateLimiter(ctrl)
	mockLogger := mock_logging.NewMockLogger(ctrl)
	mockLogger.EXPECT().Named("ratelimit.Hooks").Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().CallerSkip(ratelimitCallerSkip).Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().
		Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()
	cfg := config.MockConfigForTest(t)
	osCfg := config.NewOperationSystemConfig(cfg)
	ipCfg := config.NewIPRateLimitConfig(cfg)

	h := newRateLimitCleanupHook(rl, mockLogger, osCfg, ipCfg)

	reg.EXPECT().RegisterStart(gomock.Any())
	reg.EXPECT().RegisterStop(gomock.Any())

	h.Register(reg)
}

func Test_rateLimitCleanupHook_onStart(t *testing.T) {
	t.Parallel()

	t.Run("tickerでCleanupが呼ばれる", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		rl := mock_ratelimit.NewMockIPRateLimiter(ctrl)
		mockLogger := mock_logging.NewMockLogger(ctrl)
		mockLogger.EXPECT().Named("ratelimit.Hooks").Return(mockLogger).AnyTimes()
		mockLogger.EXPECT().CallerSkip(ratelimitCallerSkip).Return(mockLogger).AnyTimes()
		mockLogger.EXPECT().
			Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes()
		mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()

		cfg := config.MockConfigForTest(t)
		osCfg := config.NewOperationSystemConfig(cfg)
		ipCfg := config.NewIPRateLimitConfig(cfg)

		interval := 10 * time.Millisecond
		ipCfg.SetCleanupInterval(t, interval)

		done := make(chan struct{})
		rl.EXPECT().Cleanup().Do(func() { close(done) }).Times(1)

		h := newRateLimitCleanupHook(rl, mockLogger, osCfg, ipCfg)

		err := h.onStart(context.Background())
		require.NoError(t, err)

		select {
		case <-done:
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("Cleanup was not called by ticker")
		}

		require.NoError(t, h.onStop(context.Background()))
	})

	t.Run("startCtxがキャンセルされたらgoroutineが止まる", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		rl := mock_ratelimit.NewMockIPRateLimiter(ctrl)
		mockLogger := mock_logging.NewMockLogger(ctrl)
		mockLogger.EXPECT().Named("ratelimit.Hooks").Return(mockLogger).AnyTimes()
		mockLogger.EXPECT().CallerSkip(ratelimitCallerSkip).Return(mockLogger).AnyTimes()
		mockLogger.EXPECT().
			Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes()
		mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()
		rl.EXPECT().Cleanup().AnyTimes()

		cfg := config.MockConfigForTest(t)
		osCfg := config.NewOperationSystemConfig(cfg)
		ipCfg := config.NewIPRateLimitConfig(cfg)

		h := newRateLimitCleanupHook(rl, mockLogger, osCfg, ipCfg)

		ctx, cancel := context.WithCancel(context.Background())
		err := h.onStart(ctx)
		require.NoError(t, err)
		require.NotNil(t, h.stopCh)

		cancel()
		time.Sleep(10 * time.Millisecond)

		require.NoError(t, h.onStop(context.Background()))
		require.NoError(t, h.onStop(context.Background()))
	})

	t.Run("onStopでstopChを閉じるとgoroutineが止まる", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		rl := mock_ratelimit.NewMockIPRateLimiter(ctrl)
		mockLogger := mock_logging.NewMockLogger(ctrl)
		mockLogger.EXPECT().Named("ratelimit.Hooks").Return(mockLogger).AnyTimes()
		mockLogger.EXPECT().CallerSkip(ratelimitCallerSkip).Return(mockLogger).AnyTimes()
		mockLogger.EXPECT().
			Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes()
		mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()
		rl.EXPECT().Cleanup().AnyTimes()

		cfg := config.MockConfigForTest(t)
		osCfg := config.NewOperationSystemConfig(cfg)
		ipCfg := config.NewIPRateLimitConfig(cfg)

		h := newRateLimitCleanupHook(rl, mockLogger, osCfg, ipCfg)

		err := h.onStart(context.Background())
		require.NoError(t, err)
		require.NotNil(t, h.stopCh)

		require.NoError(t, h.onStop(context.Background()))
		require.NoError(t, h.onStop(context.Background()))
	})
}
