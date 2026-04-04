package recovery

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	logger := logging.NewTestLogger(t)

	require.NotNil(t, Middleware(logger, lf, appCfg))
}

func Test_newRecoverLogErrorFunc(t *testing.T) {
	t.Parallel()

	logger := logging.NewTestLogger(t)

	lf := logging.NewTestLogFieldBuilder(t)

	e := echo.New()

	t.Run("RemoteAddrがある場合、関数はnilを返しpanicしない", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/path", nil)
		req.RemoteAddr = "9.8.7.6:1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		f := newRecoverLogErrorFunc(logger, lf)
		err := f(c, fmt.Errorf("boom"), []byte("stack"))
		require.NoError(t, err)
	})

	t.Run("X-Real-Ipヘッダがある場合、関数はnilを返しpanicしない", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/other", nil)
		req.Header.Set("X-Real-Ip", "10.0.0.1")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		f := newRecoverLogErrorFunc(logger, lf)
		err := f(c, fmt.Errorf("boom2"), []byte("stack2"))
		require.NoError(t, err)
	})
}

func Test_newRecoverConfig(t *testing.T) {
	t.Parallel()

	logger := logging.NewTestLogger(t)

	t.Run("開発モードの場合、developmentConfigを返す", func(t *testing.T) {
		t.Parallel()

		appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
		appCfg.SetApplicationMode(t, config.DevelopmentMode)

		expected := developmentConfig()
		actual := newRecoverConfig(logger, appCfg)
		require.Equal(t, expected, actual)
	})

	t.Run("本番モードの場合、productionConfigを返す", func(t *testing.T) {
		t.Parallel()

		appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
		appCfg.SetApplicationMode(t, config.ProductionMode)

		expected := productionConfig()
		actual := newRecoverConfig(logger, appCfg)
		require.Equal(t, expected, actual)
	})

	t.Run("不明なモードの場合、warningを出してproductionConfigを返す", func(t *testing.T) {
		t.Parallel()

		appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
		appCfg.SetApplicationMode(t, "unknown-mode")

		expected := productionConfig()
		actual := newRecoverConfig(logger, appCfg)
		require.Equal(t, expected, actual)
	})
}

func TestDevelopmentConfig(t *testing.T) {
	t.Parallel()
	expected := middleware.RecoverConfig{
		StackSize:         10 << 10,
		DisableStackAll:   false,
		DisablePrintStack: false,
		LogLevel:          log.DEBUG,
	}

	actual := developmentConfig()

	require.Equal(t, expected, actual)
}

func TestProductionConfig(t *testing.T) {
	t.Parallel()
	expected := middleware.RecoverConfig{
		StackSize:         4 << 10,
		DisableStackAll:   true,
		DisablePrintStack: true,
		LogLevel:          log.ERROR,
	}

	actual := productionConfig()
	require.Equal(t, expected, actual)
}
