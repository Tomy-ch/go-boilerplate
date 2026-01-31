package security

import (
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack/cookie"

	"github.com/stretchr/testify/require"
)

func TestCookieModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, CookieModule())
}

func TestCookieMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCookieCfg := config.NewSecureCookieConfig(cfg)
	secCookie := cookie.NewSecurityCookie(secCookieCfg)
	require.NotNil(t, secCookie)

	out := CookieMiddleware(secCookie)

	require.NotNil(t, out.Middleware.Middleware)
}
