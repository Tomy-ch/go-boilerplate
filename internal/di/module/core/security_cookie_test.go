package core

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/cookie"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// securityCookieDeps は、SecurityCookieModule の解決に必要な設定依存を返します。
func securityCookieDeps(t *testing.T) fx.Option {
	t.Helper()

	return fx.Options(
		fx.Provide(func() testing.TB { return t }),
		fx.Provide(config.MockConfigForTest),
		fx.Provide(config.NewSecureCookieConfig),
	)
}

func TestSecurityCookieModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	var sc *cookie.SecurityCookie

	validateGraph(t, securityCookieDeps(t), SecurityCookieModule(), fx.Populate(&sc))
}

func TestSecurityCookieModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx アプリで SecurityCookie が提供される", func(t *testing.T) {
			t.Parallel()

			var sc *cookie.SecurityCookie
			app := fx.New(
				securityCookieDeps(t),
				SecurityCookieModule(),
				fx.Populate(&sc),
				fx.NopLogger,
			)

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.NotNil(t, sc)
		})
	})
}
