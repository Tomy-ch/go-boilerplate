package banner

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("本番モードの場合、バナーは非表示にする", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		cfg := &config.ApplicationConfig{}
		cfg.SetApplicationMode(t, config.ProductionMode)

		New(e, cfg)
		assert.True(t, e.HideBanner)
	})

	t.Run("開発モードの場合、バナーは表示される", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		cfg := &config.ApplicationConfig{}
		cfg.SetApplicationMode(t, config.DevelopmentMode)

		New(e, cfg)
		assert.False(t, e.HideBanner)
	})
}
