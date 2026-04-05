package shutdowner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func Test_ModuleProvidesRegistrar(t *testing.T) {
	t.Parallel()

	var got Shutdowner

	app := fx.New(
		Module(),
		fx.Invoke(func(r Shutdowner) {
			got = r
		}),
	)

	require.NotNil(t, app)
	require.NoError(t, app.Start(context.Background()))
	defer func() { _ = app.Stop(context.Background()) }()

	require.NotNil(t, got)
}
