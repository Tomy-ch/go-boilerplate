package instrumentation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/server/extension"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveThroughMiddleware は mw 適用後に / を1回叩き応答コードを返す。ミドルウェアがリクエスト
// チェーンを壊さず素通しさせることを検証するために用いる。
func serveThroughMiddleware(t *testing.T, mw echo.MiddlewareFunc) int {
	t.Helper()
	e := echo.New()
	e.Use(mw)
	e.GET("/", func(c *echo.Context) error { return c.NoContent(http.StatusNoContent) })
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec.Code
}

// requireObservabilityMiddleware は DI 層の責務である配線（Priority / Name / 非nil）と、
// 生成されたミドルウェアが素通しで機能することを検証する。echootel と素通しの挙動差自体は
// observability パッケージ側のテストが担う。
func requireObservabilityMiddleware(t *testing.T, out extension.UseMiddlewareOut) {
	t.Helper()
	assert.Equal(t, observabilityPriority, out.Middleware.Priority)
	assert.Equal(t, "observability", out.Middleware.Name)
	require.NotNil(t, out.Middleware.Middleware)
	assert.Equal(t, http.StatusNoContent, serveThroughMiddleware(t, out.Middleware.Middleware))
}

func TestObservabilityMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("可観測が有効ならOTelミドルウェアを配線し素通しさせる", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			obsCfg := config.NewObservabilityConfig(cfg)

			requireObservabilityMiddleware(t, ObservabilityMiddleware(obsCfg))
		})

		t.Run("可観測が無効なら素通しミドルウェアを配線し素通しさせる", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			obsCfg := config.NewObservabilityConfig(cfg)
			obsCfg.SetObservabilityTracesExporter(t, "")
			obsCfg.SetObservabilityMetricsExporter(t, "")

			requireObservabilityMiddleware(t, ObservabilityMiddleware(obsCfg))
		})
	})
}
