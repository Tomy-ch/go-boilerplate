package server

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAppServer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Echoインスタンスを生成する", func(t *testing.T) {
			t.Parallel()

			assert.NotNil(t, NewAppServer())
		})
	})
}

func TestNewHTTPServer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Echoをハンドラとし設定値を反映したhttp.Serverを生成する", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			srvCfg := config.NewServerConfig(cfg)
			e := NewAppServer()

			actual := NewHTTPServer(e, srvCfg)
			require.NotNil(t, actual)

			assert.Same(t, e, actual.Handler)
			assert.Equal(t, srvCfg.ReadHeaderTimeout(), actual.ReadHeaderTimeout)
			assert.Equal(t, srvCfg.ReadTimeout(), actual.ReadTimeout)
			assert.Equal(t, srvCfg.WriteTimeout(), actual.WriteTimeout)
			assert.Equal(t, srvCfg.IdleTimeout(), actual.IdleTimeout)
		})
	})
}
