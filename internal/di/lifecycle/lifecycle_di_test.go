package lifecycle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func Test_ModuleProvidesRegistrar(t *testing.T) {
	t.Parallel()

	var got Registrar

	app := fx.New(
		Module(),
		fx.Invoke(func(r Registrar) {
			got = r
		}),
	)

	require.NotNil(t, app)
	// Start the app so fx.Invoke runs and provides the Registrar
	require.NoError(t, app.Start(context.Background()))
	defer func() { _ = app.Stop(context.Background()) }()

	require.NotNil(t, got)
}
