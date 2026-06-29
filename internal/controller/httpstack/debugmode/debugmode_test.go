package debugmode

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	exec := func(t *testing.T, mode string) bool {
		t.Helper()
		e := echo.New()
		appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
		appCfg.SetApplicationMode(t, mode)
		New(e, appCfg)
		return e.Debug
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("開発モードの場合、Debugがtrueになる", func(t *testing.T) {
			t.Parallel()
			assert.True(t, exec(t, config.DevelopmentMode))
		})

		t.Run("開発モード以外の場合、Debugがfalseになる", func(t *testing.T) {
			t.Parallel()
			assert.False(t, exec(t, config.ProductionMode))
		})
	})
}
