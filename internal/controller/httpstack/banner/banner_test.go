package banner

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
		cfg := &config.ApplicationConfig{}
		cfg.SetApplicationMode(t, mode)
		New(e, cfg)
		return e.HideBanner
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本番モードの場合、バナーは非表示にする", func(t *testing.T) {
			t.Parallel()
			assert.True(t, exec(t, config.ProductionMode))
		})

		t.Run("開発モードの場合、バナーは表示される", func(t *testing.T) {
			t.Parallel()
			assert.False(t, exec(t, config.DevelopmentMode))
		})
	})
}
