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
}

func Test_buildCORSConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		allowedOrigins []string
	}{
		{"複数Originがそのまま反映される", []string{"http://localhost:3000", "http://localhost:4000"}},
		{"nilがそのまま反映される", nil},
		{"空スライスがそのまま反映される", []string{}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := buildCORSConfig(tt.allowedOrigins)
			assert.Equal(t, tt.allowedOrigins, cfg.AllowOrigins)
			assert.False(t, cfg.AllowCredentials)
			assert.Equal(t, corsMaxAgeSeconds, cfg.MaxAge)
		})
	}
}
