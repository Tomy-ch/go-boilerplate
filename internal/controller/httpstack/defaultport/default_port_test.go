package defaultport

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		mode         string
		wantHidePort bool
	}{
		{"本番モードの場合、ポートは非表示にする", config.ProductionMode, true},
		{"開発モードの場合、ポートは表示される", config.DevelopmentMode, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
			appCfg.SetApplicationMode(t, tt.mode)

			New(e, appCfg)
			assert.Equal(t, tt.wantHidePort, e.HidePort)
		})
	}
}
