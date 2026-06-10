package debugmode

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		mode      string
		wantDebug bool
	}{
		{"開発モードの場合、Debugがtrueになること", config.DevelopmentMode, true},
		{"開発モード以外の場合、Debugがfalseになること", config.ProductionMode, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
			appCfg.SetApplicationMode(t, tt.mode)

			New(e, appCfg)
			assert.Equal(t, tt.wantDebug, e.Debug)
		})
	}
}
