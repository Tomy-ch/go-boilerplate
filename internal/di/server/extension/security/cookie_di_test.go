package security

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/cookie"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCookieMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCookieCfg := config.NewSecureCookieConfig(cfg)
	secCookie := cookie.NewSecurityCookie(secCookieCfg)
	require.NotNil(t, secCookie)

	out := CookieMiddleware(secCookie)

	assert.Equal(t, "secure_cookie", out.Middleware.Name)
	assert.Equal(t, cookiePriority, out.Middleware.Priority)
	assert.NotNil(t, out.Middleware.Middleware)
}
