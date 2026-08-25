package module

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/authz/allowall"
	"go-boilerplate/internal/logging"
	authzbd "go-boilerplate/internal/usecase/boundary/authz"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func Test_authzModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// Authorizer の配線のみを検証する。ApplicationConfig / Logger は commonDeps が供給する。
	opts := append(commonDeps(),
		authzModule(),
	)
	validateGraph(t, opts...)
}

func Test_provideAuthorizer(t *testing.T) {
	t.Parallel()

	newAppCfg := func(t *testing.T, env string) *config.ApplicationConfig {
		t.Helper()
		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		appCfg.SetApplicationEnv(t, env)

		return appCfg
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ローカル環境では全許可Authorizerが提供されWARNが出る", func(t *testing.T) {
			t.Parallel()
			logger, logs := logging.NewObservedTestLogger(t)
			authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvLocal), Logger: logger})
			require.NoError(t, err)
			expected, err := allowall.New(newAppCfg(t, config.EnvLocal))
			require.NoError(t, err)
			assert.Equal(t, expected, authorizer)
			assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		})

		t.Run("dast環境では全許可Authorizerが提供されWARNが出る", func(t *testing.T) {
			t.Parallel()
			logger, logs := logging.NewObservedTestLogger(t)
			authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvDast), Logger: logger})
			require.NoError(t, err)
			expected, err := allowall.New(newAppCfg(t, config.EnvDast))
			require.NoError(t, err)
			assert.Equal(t, expected, authorizer)
			assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		})

		t.Run("CI環境では全許可Authorizerが提供されWARNが出る", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)

			authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvCI), Logger: logger})
			require.NoError(t, err)
			expected, err := allowall.New(newAppCfg(t, config.EnvCI))
			require.NoError(t, err)
			assert.Equal(t, expected, authorizer)
			assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		})

		t.Run("テスト環境では全許可Authorizerが提供されWARNが出る", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)

			authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvTest), Logger: logger})
			require.NoError(t, err)
			expected, err := allowall.New(newAppCfg(t, config.EnvTest))
			require.NoError(t, err)
			assert.Equal(t, expected, authorizer)
			assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// local / ci / test 以外で、対応する認可実装が配線されていない環境は
		// すべて fail-closed（起動エラー）になること。
		failClosedEnvs := []string{
			config.EnvDevelopment,
			config.EnvStaging,
			config.EnvProduction,
			"unknown-env",
		}

		for _, env := range failClosedEnvs {
			t.Run(env+"では認可を配線せずエラーを返す", func(t *testing.T) {
				t.Parallel()

				logger := logging.NewTestLogger(t)

				authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, env), Logger: logger})
				require.ErrorIs(t, err, errNoAuthorizerForEnv)
				require.ErrorContains(t, err, env)
				assert.Nil(t, authorizer)
			})
		}
	})
}

func Test_authzModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecase 層が参照する Authorizer を提供する", func(t *testing.T) {
			t.Parallel()

			var authorizer authzbd.Authorizer

			opts := append(commonDeps(),
				authzModule(),
				fx.Populate(&authorizer),
			)
			validateGraph(t, opts...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では Authorizer が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var authorizer authzbd.Authorizer

			opts := append(commonDeps(), fx.Populate(&authorizer), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
