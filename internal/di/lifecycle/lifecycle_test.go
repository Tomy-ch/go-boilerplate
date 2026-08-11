package lifecycle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestNewLifecycleRegistrar(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx.Lifecycleから非nilなRegistrarを構築する", func(t *testing.T) {
			t.Parallel()

			var got Registrar
			app := fx.New(
				fx.Invoke(func(lc fx.Lifecycle) { got = NewLifecycleRegistrar(lc) }),
				fx.NopLogger,
			)
			require.NoError(t, app.Err())
			assert.NotNil(t, got)
		})
	})
}

func Test_lifecycleRegistrar_RegisterStart(t *testing.T) {
	t.Parallel()

	var started bool

	app := fx.New(
		fx.Provide(NewLifecycleRegistrar),
		fx.Invoke(func(r Registrar) {
			r.RegisterStart(func(_ context.Context) error {
				started = true
				return nil
			})
		}),
		fx.NopLogger,
	)

	require.NoError(t, app.Start(context.Background()))
	defer func() { _ = app.Stop(context.Background()) }()

	assert.True(t, started)
}

func Test_lifecycleRegistrar_RegisterStop(t *testing.T) {
	t.Parallel()

	var stopped bool

	app := fx.New(
		fx.Provide(NewLifecycleRegistrar),
		fx.Invoke(func(r Registrar) {
			r.RegisterStop(func(_ context.Context) error {
				stopped = true
				return nil
			})
		}),
		fx.NopLogger,
	)

	require.NoError(t, app.Start(context.Background()))
	require.NoError(t, app.Stop(context.Background()))

	assert.True(t, stopped)
}
