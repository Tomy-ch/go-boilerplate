package banner

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		mode     string
		wantHide bool
	}{
		{name: "本番モードの場合、バナーは非表示にする", mode: config.ProductionMode, wantHide: true},
		{name: "開発モードの場合、バナーは表示される", mode: config.DevelopmentMode, wantHide: false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			cfg := &config.ApplicationConfig{}
			cfg.SetApplicationMode(t, tt.mode)

			New(e, cfg)
			assert.Equal(t, tt.wantHide, e.HideBanner)
		})
	}
}
