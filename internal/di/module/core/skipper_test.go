package core

import (
	"context"
	"testing"

	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestSkipperModule(t *testing.T) {
	t.Run("fx アプリで Skipper が提供される", func(t *testing.T) {
		var s echomw.Skipper
		app := fx.New(
			SkipperModule(),
			fx.Populate(&s),
		)

		require.NoError(t, app.Start(context.Background()))
		require.NotNil(t, s)
		require.NotPanics(t, func() { _ = s })
		require.NoError(t, app.Stop(context.Background()))
	})
}
