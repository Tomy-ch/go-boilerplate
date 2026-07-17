package nonprod

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_provideDebugModeServeConfig(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("開発モードではDebugが有効になる", func(t *testing.T) {
			t.Parallel()

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
			appCfg.SetApplicationMode(t, config.DevelopmentMode)

			out := provideDebugModeServeConfig(appCfg)
			require.NotNil(t, out.SrvCfg.Config)
			assert.Equal(t, "debugmode", out.SrvCfg.Name)

			e := &echo.Echo{}
			out.SrvCfg.Config(e)
			assert.True(t, e.Debug)
		})

		t.Run("本番モードではDebugが無効のままになる", func(t *testing.T) {
			t.Parallel()

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
			appCfg.SetApplicationMode(t, config.ProductionMode)

			out := provideDebugModeServeConfig(appCfg)
			require.NotNil(t, out.SrvCfg.Config)
			assert.Equal(t, "debugmode", out.SrvCfg.Name)

			e := &echo.Echo{}
			out.SrvCfg.Config(e)
			assert.False(t, e.Debug)
		})
	})
}
