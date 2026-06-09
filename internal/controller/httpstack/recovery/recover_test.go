package recovery

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/errorhandler"
	"go-boilerplate/internal/controller/server"
	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/assert"
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

	t.Run("RemoteAddrがある場合、元errを返しリカバリ済みを記録する", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/path", nil)
		req.RemoteAddr = "9.8.7.6:1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		inErr := fmt.Errorf("boom")
		f := newRecoverLogErrorFunc(logger, lf)
		err := f(c, inErr, []byte("stack"))
		require.ErrorIs(t, err, inErr)
		assert.True(t, server.IsRecovered(c))
	})

	t.Run("X-Real-Ipヘッダがある場合、元errを返しリカバリ済みを記録する", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/other", nil)
		req.Header.Set("X-Real-Ip", "10.0.0.1")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		inErr := fmt.Errorf("boom2")
		f := newRecoverLogErrorFunc(logger, lf)
		err := f(c, inErr, []byte("stack2"))
		require.ErrorIs(t, err, inErr)
		assert.True(t, server.IsRecovered(c))
	})
}

func TestMiddleware_realPanic(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	obsLogger, observed := logging.NewObservedTestLogger(t)

	e := echo.New()
	e.Use(Middleware(obsLogger, lf, appCfg))
	e.GET("/panic", func(_ echo.Context) error { panic("boom-panic") })

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/panic", nil)
	e.ServeHTTP(httptest.NewRecorder(), req)

	entries := observed.FilterMessage("panic recovered").All()
	require.Len(t, entries, 1)
	cm := entries[0].ContextMap()

	errStr, ok := cm[logging.InternalErrorKey].(string)
	require.True(t, ok)
	assert.Contains(t, errStr, "boom-panic")

	// 実ランタイムスタックが可読文字列(Base64 でない)で出力されること。
	stackStr, ok := cm[logging.InternalStackTraceKey].(string)
	require.True(t, ok)
	assert.Contains(t, stackStr, "goroutine")
}

func TestMiddleware_panicReturns500WithSingleLog(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	obsLogger, observed := logging.NewObservedTestLogger(t)

	e := echo.New()
	errorhandler.New(e, obsLogger, lf, obsCfg)
	e.Use(Middleware(obsLogger, lf, appCfg))
	e.GET("/panic", func(_ echo.Context) error { panic("boom-panic") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/panic", nil)
	e.ServeHTTP(rec, req)

	// パニックでも 200 空ではなく 500 が返る。
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	// ログは middleware.recover の 1 件のみ（エラーハンドラの重複ログは出ない）。
	assert.Equal(t, 1, observed.Len())
	assert.Equal(t, 1, observed.FilterMessage("panic recovered").Len())
	assert.Equal(t, 0, observed.FilterMessage("errorhandler.server_error").Len())
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
		assert.Equal(t, expected, actual)
	})

	t.Run("本番モードの場合、productionConfigを返す", func(t *testing.T) {
		t.Parallel()

		appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
		appCfg.SetApplicationMode(t, config.ProductionMode)

		expected := productionConfig()
		actual := newRecoverConfig(logger, appCfg)
		assert.Equal(t, expected, actual)
	})

	t.Run("不明なモードの場合、warningを出してproductionConfigを返す", func(t *testing.T) {
		t.Parallel()

		appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
		appCfg.SetApplicationMode(t, "unknown-mode")

		expected := productionConfig()
		actual := newRecoverConfig(logger, appCfg)
		assert.Equal(t, expected, actual)
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

	assert.Equal(t, expected, actual)
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
	assert.Equal(t, expected, actual)
}
