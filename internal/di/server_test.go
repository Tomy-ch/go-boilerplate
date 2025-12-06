package di

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func Test_NewApplicationCore_ReturnsApp(t *testing.T) {
	t.Parallel()

	app := NewApplicationCore()
	require.NotNil(t, app)
}

func Test_NewApplicationServer_WrapsAppStartStop(t *testing.T) {
	t.Parallel()

	app := fx.New()
	defer func() {
		_ = app.Stop(context.Background())
	}()

	start, stop := NewApplicationServer(app)
	require.NotNil(t, start)
	require.NotNil(t, stop)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, start(ctx))

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	require.NoError(t, stop(stopCtx))
}
