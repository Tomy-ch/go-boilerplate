package debugmode

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("開発モードの場合、Debugがtrueになること", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		cfg := &config.Config{}
		cfg.SetServerAppMode(t, config.DevelopmentMode)

		New(e, cfg)
		require.True(t, e.Debug)
	})

	t.Run("開発モード以外の場合、Debugがfalseになること", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		cfg := &config.Config{}
		cfg.SetServerAppMode(t, config.ProductionMode)

		New(e, cfg)
		require.False(t, e.Debug)
	})
}
