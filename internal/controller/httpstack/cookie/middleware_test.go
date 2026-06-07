package cookie

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	next := func(c echo.Context) error {
		// Add a Set-Cookie header that should be rewritten
		c.Response().Header().Add("Set-Cookie", "sessionid=abc; Path=/; HttpOnly")
		return c.String(http.StatusOK, "ok")
	}

	handler := secureCookieMiddleware(secCookie)(next)
	require.NoError(t, handler(c))

	cookies := rec.Header().Values("Set-Cookie")
	require.NotEmpty(t, cookies)

	// Expect rewrite to include SameSite and Domain from mock config
	foundSameSite := false
	foundDomain := false
	foundSecure := false
	for _, raw := range cookies {
		if strings.Contains(raw, "SameSite=") {
			foundSameSite = true
		}
		if strings.Contains(raw, "Domain=") {
			foundDomain = true
		}
		if strings.Contains(raw, "Secure") {
			foundSecure = true
		}
	}

	assert.True(t, foundSameSite, "SameSite must be present in rewritten Set-Cookie")
	assert.True(t, foundDomain, "Domain must be present in rewritten Set-Cookie")
	assert.True(t, foundSecure, "Secure flag must be present in rewritten Set-Cookie")
}
