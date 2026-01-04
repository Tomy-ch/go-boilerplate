package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"boilerplate-go/internal/config"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestValidatorModule_ProvidesValidator(t *testing.T) {
	t.Run("fx アプリで Validator が提供される", func(t *testing.T) {
		var v *openapi3.T
		app := fx.New(
			fx.Provide(func() testing.TB { return t }),
			ValidatorModule(),
			fx.Provide(config.MockConfigForTest),
			fx.Populate(&v),
			fx.NopLogger,
		)

		require.NoError(t, app.Start(context.Background()))
		require.NotNil(t, v)
		require.NotPanics(t, func() { _ = v })
		require.NoError(t, app.Stop(context.Background()))
	})
}
