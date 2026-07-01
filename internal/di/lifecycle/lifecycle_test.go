package lifecycle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func Test_RegisterStartExecutesOnAppStart(t *testing.T) {
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

func Test_RegisterShutdownExecutesOnAppStop(t *testing.T) {
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

	// start and then stop to trigger OnStop hooks
	require.NoError(t, app.Start(context.Background()))
	require.NoError(t, app.Stop(context.Background()))

	assert.True(t, stopped)
}
