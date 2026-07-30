package core

import (
	"context"
	"testing"

	echomw "github.com/labstack/echo/v5/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestSkipperModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx アプリで Skipper が提供される", func(t *testing.T) {
			t.Parallel()

			var s echomw.Skipper
			app := fx.New(
				SkipperModule(),
				fx.Populate(&s),
			)

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.NotNil(t, s)
		})
	})
}
