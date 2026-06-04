package shutdowner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func Test_ModuleProvidesWorkingShutdowner(t *testing.T) {
	t.Parallel()

	// Module() が提供する Shutdowner が実 fx.Shutdowner に結線されており、
	// Shutdown() がアプリへ伝播して app.Wait() にシャットダウンシグナルが届くことを検証する。
	var sd Shutdowner

	app := fx.New(
		Module(),
		fx.Invoke(func(s Shutdowner) { sd = s }),
		fx.NopLogger,
	)

	require.NoError(t, app.Start(context.Background()))
	defer func() { _ = app.Stop(context.Background()) }()

	require.NotNil(t, sd)
	require.NoError(t, sd.Shutdown())

	select {
	case <-app.Wait():
		// Shutdown 要求がアプリへ伝播した
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown が app.Wait() に伝播しなかった")
	}
}
