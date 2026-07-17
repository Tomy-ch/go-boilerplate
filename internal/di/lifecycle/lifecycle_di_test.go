package lifecycle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestModule(t *testing.T) {
	t.Parallel()

	// Module() が提供する Registrar に登録した start/stop が、app の起動/停止で実際に発火することを検証する。
	var started, stopped bool

	app := fx.New(
		Module(),
		fx.Invoke(func(r Registrar) {
			r.RegisterStart(func(context.Context) error { started = true; return nil })
			r.RegisterStop(func(context.Context) error { stopped = true; return nil })
		}),
		fx.NopLogger,
	)

	require.NoError(t, app.Start(context.Background()))
	assert.True(t, started, "Module が提供する Registrar の start フックが発火すること")

	require.NoError(t, app.Stop(context.Background()))
	assert.True(t, stopped, "Module が提供する Registrar の stop フックが発火すること")
}
