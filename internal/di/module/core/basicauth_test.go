package core

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"

	echomw "github.com/labstack/echo/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// basicAuthDeps は、BasicAuthModule の解決に必要な設定依存を返します。
func basicAuthDeps(t *testing.T) fx.Option {
	t.Helper()

	return fx.Options(
		fx.Provide(func() testing.TB { return t }),
		fx.Provide(config.MockConfigForTest),
		fx.Provide(config.NewMetricsConfig),
	)
}

func TestBasicAuthModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	var b echomw.BasicAuthValidator

	validateGraph(t, basicAuthDeps(t), BasicAuthModule(), fx.Populate(&b))
}

func TestBasicAuthModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx アプリで BasicAuthValidator が提供される", func(t *testing.T) {
			t.Parallel()

			var b echomw.BasicAuthValidator
			app := fx.New(
				basicAuthDeps(t),
				BasicAuthModule(),
				fx.Populate(&b),
			)

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.NotNil(t, b)
		})
	})
}
