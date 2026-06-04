package debugmode

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("開発モードの場合、Debugがtrueになること", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
		appCfg.SetApplicationMode(t, config.DevelopmentMode)

		New(e, appCfg)
		assert.True(t, e.Debug)
	})

	t.Run("開発モード以外の場合、Debugがfalseになること", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
		appCfg.SetApplicationMode(t, config.ProductionMode)

		New(e, appCfg)
		assert.False(t, e.Debug)
	})
}
