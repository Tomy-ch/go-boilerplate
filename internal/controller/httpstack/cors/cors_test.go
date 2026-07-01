package cors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("許可オリジンからのプリフライトでAccess-Control-Allow-Originが付与される", func(t *testing.T) {
			t.Parallel()

			secCfg := config.NewSecurityConfig(config.MockConfigForTest(t))
			e := echo.New()
			e.Use(Middleware(secCfg))
			e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

			origin := secCfg.AllowedOrigins()[0]
			req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/", nil)
			req.Header.Set(echo.HeaderOrigin, origin)
			req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, origin, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不許可オリジンからのプリフライトではAccess-Control-Allow-Originが付与されない", func(t *testing.T) {
			t.Parallel()

			secCfg := config.NewSecurityConfig(config.MockConfigForTest(t))
			e := echo.New()
			e.Use(Middleware(secCfg))
			e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

			req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/", nil)
			req.Header.Set(echo.HeaderOrigin, "https://evil.example.com")
			req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Empty(t, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
		})
	})
}

func Test_buildCORSConfig(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数Originがそのまま反映される", func(t *testing.T) {
			t.Parallel()
			origins := []string{"http://localhost:3000", "http://localhost:4000"}
			cfg := buildCORSConfig(origins)
			assert.Equal(t, origins, cfg.AllowOrigins)
			assert.Equal(t, []string{echo.HEAD, echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS}, cfg.AllowMethods)
			assert.Equal(t, []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization}, cfg.AllowHeaders)
			assert.Equal(t, []string{echo.HeaderContentDisposition, echo.HeaderLocation, echo.HeaderXRequestID}, cfg.ExposeHeaders)
			assert.False(t, cfg.AllowCredentials)
			assert.Equal(t, corsMaxAgeSeconds, cfg.MaxAge)
		})

		t.Run("nilがそのまま反映される", func(t *testing.T) {
			t.Parallel()
			cfg := buildCORSConfig(nil)
			assert.Nil(t, cfg.AllowOrigins)
			assert.False(t, cfg.AllowCredentials)
			assert.Equal(t, corsMaxAgeSeconds, cfg.MaxAge)
		})

		t.Run("空スライスがそのまま反映される", func(t *testing.T) {
			t.Parallel()
			cfg := buildCORSConfig([]string{})
			assert.Equal(t, []string{}, cfg.AllowOrigins)
			assert.False(t, cfg.AllowCredentials)
			assert.Equal(t, corsMaxAgeSeconds, cfg.MaxAge)
		})
	})
}
