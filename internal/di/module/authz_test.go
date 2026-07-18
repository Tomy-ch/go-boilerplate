package module

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/authz/allowall"
	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_authzModule_GraphIsValid(t *testing.T) {
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
			expected, err := allowall.New(appCfg)
			require.NoError(t, err)
			assert.Equal(t, expected, authorizer)
			// 全許可スタブ配線時に WARN で注意喚起されること。
			assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		})

		t.Run("CI環境では全許可Authorizerが提供されWARNが出る", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			appCfg.SetApplicationEnv(t, config.EnvCI)
			logger, logs := logging.NewObservedTestLogger(t)

			authorizer, err := provideAuthorizer(appCfg, logger)
			require.NoError(t, err)
			expected, err := allowall.New(appCfg)
			require.NoError(t, err)
			assert.Equal(t, expected, authorizer)
			assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		})

		t.Run("テスト環境では全許可Authorizerが提供されWARNが出る", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			appCfg.SetApplicationEnv(t, config.EnvTest)
			logger, logs := logging.NewObservedTestLogger(t)

			authorizer, err := provideAuthorizer(appCfg, logger)
			require.NoError(t, err)
			expected, err := allowall.New(appCfg)
			require.NoError(t, err)
			assert.Equal(t, expected, authorizer)
			assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// local / ci / test 以外（本番相当）はすべて fail-closed でエラーを返すこと。
		assertFailClosed := func(t *testing.T, env string) {
			t.Helper()
			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			appCfg.SetApplicationEnv(t, env)
			logger := logging.NewTestLogger(t)

			authorizer, err := provideAuthorizer(appCfg, logger)
			require.Error(t, err)
			assert.Nil(t, authorizer)
		}

		t.Run("development環境では全許可を配線せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			assertFailClosed(t, config.EnvDevelopment)
		})

		t.Run("staging環境では全許可を配線せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			assertFailClosed(t, config.EnvStaging)
		})

		t.Run("production環境では全許可を配線せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			assertFailClosed(t, config.EnvProduction)
		})
	})
}
