package module

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/system"
)

func TestSystemModule_ProvidesBuildInfo(t *testing.T) {
	t.Parallel()

	t.Run("fx アプリで BuildInfo が提供される", func(t *testing.T) {
		t.Parallel()

		var bi system.BuildInfo

		app := fx.New(
			SystemModule(),
			fx.Populate(&bi),
			fx.NopLogger,
		)

		require.NoError(t, app.Start(context.Background()))
		require.NotNil(t, bi)
		// Methods should be callable
		_ = bi.Version()
		_ = bi.Revision()
		_ = bi.BuildDate()
		require.NoError(t, app.Stop(context.Background()))
	})
}
