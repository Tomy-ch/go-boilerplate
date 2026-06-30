package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/config"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestValidatorModule_ProvidesValidator(t *testing.T) {
	t.Parallel()

	t.Run("fx アプリで Validator が提供される", func(t *testing.T) {
		t.Parallel()

		var v *openapi3.T
		app := fx.New(
			fx.Provide(func() testing.TB { return t }),
			ValidatorModule(),
			fx.Provide(config.MockConfigForTest),
			fx.Populate(&v),
			fx.NopLogger,
		)

		require.NoError(t, app.Start(context.Background()))
		t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
		assert.NotNil(t, v)
	})
}
