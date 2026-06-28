package module

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/authz/allowall"
	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthzModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// Authorizer の配線のみを検証する。ApplicationConfig / Logger は commonDeps が供給する。
	opts := append(commonDeps(), authzModule())
	validateGraph(t, opts...)
}

func Test_provideAuthorizer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ローカル環境では全許可Authorizerが提供されWARNが出る", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			appCfg.SetApplicationEnv(t, config.EnvLocal)
			logger, logs := logging.NewObservedTestLogger(t)

			authorizer, err := provideAuthorizer(appCfg, logger)
			require.NoError(t, err)
			assert.Equal(t, allowall.New(), authorizer)
			// 全許可スタブ配線時に WARN で注意喚起されること。
			assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		})

		t.Run("CI環境では全許可Authorizerが提供される", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			appCfg.SetApplicationEnv(t, config.EnvCI)
			logger := logging.NewTestLogger(t)

			authorizer, err := provideAuthorizer(appCfg, logger)
			require.NoError(t, err)
			assert.Equal(t, allowall.New(), authorizer)
		})

		t.Run("テスト環境では全許可Authorizerが提供される", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			appCfg.SetApplicationEnv(t, config.EnvTest)
			logger := logging.NewTestLogger(t)

			authorizer, err := provideAuthorizer(appCfg, logger)
			require.NoError(t, err)
			assert.Equal(t, allowall.New(), authorizer)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本番相当の環境では全許可を配線せずエラーを返す", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			appCfg.SetApplicationEnv(t, config.EnvProduction)
			logger := logging.NewTestLogger(t)

			authorizer, err := provideAuthorizer(appCfg, logger)
			require.Error(t, err)
			require.Nil(t, authorizer)
		})
	})
}
