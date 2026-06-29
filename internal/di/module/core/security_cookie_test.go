package core

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/cookie"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestSecurityCookieModule_ProvidesSecurityCookie(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx アプリで SecurityCookie が提供される", func(t *testing.T) {
			t.Parallel()

			var sc *cookie.SecurityCookie
			app := fx.New(
				fx.Provide(func() testing.TB { return t }),
				fx.Provide(config.MockConfigForTest),
				fx.Provide(config.NewSecureCookieConfig),
				SecurityCookieModule(),
				fx.Populate(&sc),
				fx.NopLogger,
			)

			require.NoError(t, app.Start(context.Background()))
			require.NotNil(t, sc)
			require.NotPanics(t, func() { _ = sc })
			require.NoError(t, app.Stop(context.Background()))
		})
	})
}
