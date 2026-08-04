package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/observability"
)

// newAppCfgForEnv は、指定 env のアプリケーション設定を返します。
func newAppCfgForEnv(t *testing.T, env string) *config.ApplicationConfig {
	t.Helper()
	appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
	appCfg.SetApplicationEnv(t, env)

	return appCfg
}

func Test_allowPrivateNetworkForEnv(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("local / CI / Test は private network を許可する", func(t *testing.T) {
			t.Parallel()
			assert.True(t, allowPrivateNetworkForEnv(config.EnvLocal))
			assert.True(t, allowPrivateNetworkForEnv(config.EnvCI))
			assert.True(t, allowPrivateNetworkForEnv(config.EnvTest))
		})

		t.Run("dev / stg / prd および未知環境は許可しない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, allowPrivateNetworkForEnv(config.EnvDevelopment))
			assert.False(t, allowPrivateNetworkForEnv(config.EnvStaging))
			assert.False(t, allowPrivateNetworkForEnv(config.EnvProduction))
			assert.False(t, allowPrivateNetworkForEnv(""))
		})
	})
}

func Test_provideOutboundHTTPClient(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("env に応じた private network 方針でクライアントを構築する", func(t *testing.T) {
			t.Parallel()

			transport := observability.NewNoopHTTPClientTransport(t)

			for _, env := range []string{config.EnvLocal, config.EnvProduction} {
				got := provideOutboundHTTPClient(transport, newAppCfgForEnv(t, env))

				require.NotNil(t, got)
				require.NotNil(t, got.Client)
			}
		})
	})
}
