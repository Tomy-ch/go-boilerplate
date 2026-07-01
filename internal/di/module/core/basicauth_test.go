package core

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"

	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestBasicAuthModule(t *testing.T) {
	t.Parallel()

	t.Run("fx アプリで BasicAuthValidator が提供される", func(t *testing.T) {
		t.Parallel()

		var b echomw.BasicAuthValidator
		app := fx.New(
			fx.Provide(func() testing.TB { return t }),
			fx.Provide(config.MockConfigForTest),
			fx.Provide(config.NewMetricsConfig),
			BasicAuthModule(),
			fx.Populate(&b),
		)

		require.NoError(t, app.Start(context.Background()))
		t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
		assert.NotNil(t, b)
	})
}
