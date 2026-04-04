package defaultport

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("本番モードの場合、ポートは非表示にする", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
		appCfg.SetApplicationMode(t, config.ProductionMode)

		New(e, appCfg)
		require.True(t, e.HidePort)
	})

	t.Run("開発モードの場合、ポートは表示される", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
		appCfg.SetApplicationMode(t, config.DevelopmentMode)

		New(e, appCfg)
		require.False(t, e.HidePort)
	})
}
