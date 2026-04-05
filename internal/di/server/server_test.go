package server

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func Test_Module(t *testing.T) {
	t.Parallel()

	t.Run("Module が nil でないこと", func(t *testing.T) {
		t.Parallel()
		opt := Module()
		require.NotNil(t, opt)
	})

	t.Run("Module を fx.New に渡せること", func(t *testing.T) {
		t.Parallel()
		app := fx.New(Module())
		require.NotNil(t, app)
	})
}

func Test_HookModule(t *testing.T) {
	t.Parallel()

	t.Run("HookModule が nil でないこと", func(t *testing.T) {
		t.Parallel()
		opt := HookModule()
		require.NotNil(t, opt)
	})

	t.Run("HookModule を fx.New に渡せること", func(t *testing.T) {
		t.Parallel()
		app := fx.New(HookModule())
		require.NotNil(t, app)
	})
}

func Test_MiddlewareModule(t *testing.T) {
	t.Parallel()

	t.Run("MiddlewareModule が nil でないこと", func(t *testing.T) {
		t.Parallel()
		opt := MiddlewareModule()
		require.NotNil(t, opt)
	})

	t.Run("MiddlewareModule を fx.New に渡せること", func(t *testing.T) {
		t.Parallel()
		app := fx.New(MiddlewareModule())
		require.NotNil(t, app)
	})
}
