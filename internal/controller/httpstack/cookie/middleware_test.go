package cookie

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	require.NotNil(t, Middleware(&SecurityCookie{}))
}

func Test_secureCookieMiddleware_RewritesSetCookie(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecureCookieConfig(cfg)
	secCookie := NewSecurityCookie(secCfg)

	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	next := func(c echo.Context) error {
		// 書き換え対象となる Set-Cookie を付与する
		c.Response().Header().Add("Set-Cookie", "sessionid=abc; Path=/; HttpOnly")
		return c.String(http.StatusOK, "ok")
	}

	handler := secureCookieMiddleware(secCookie)(next)
	require.NoError(t, handler(c))

	cookies := rec.Header().Values("Set-Cookie")
	require.Len(t, cookies, 1)
	// mock config 由来の値（SameSite=Strict / Domain=localhost / Secure）が反映される
	assert.Contains(t, cookies[0], "SameSite=Strict")
	assert.Contains(t, cookies[0], "Domain=localhost")
	assert.Contains(t, cookies[0], "Secure")
}

func Test_secureCookieMiddleware_KeepsWrapperOnError(t *testing.T) {
	t.Parallel()

	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := secureCookieMiddleware(&SecurityCookie{applyToAll: true})(func(_ echo.Context) error {
		return echo.NewHTTPError(http.StatusInternalServerError)
	})
	require.Error(t, handler(c))

	// next がエラーでも Writer は復元されない（エラー経路の Set-Cookie も書き換え対象にするため）
	_, ok := c.Response().Writer.(*cookieRewriteWriter)
	assert.True(t, ok)
}
