package module

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/domain/user"                // sample-api:line
	mock_user "go-boilerplate/internal/domain/user/mock" // sample-api:line
	"go-boilerplate/internal/infrastructure/authz/allowall"
	"go-boilerplate/internal/infrastructure/authz/userrole" // sample-api:line
	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"          // sample-api:line
	"go.uber.org/mock/gomock" // sample-api:line
)

func Test_authzModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// Authorizer の配線のみを検証する。ApplicationConfig / Logger は commonDeps が供給する。
	opts := append(commonDeps(),
		fx.Provide(func() user.RoleRepository { return nil }), // sample-api:line
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

		// sample-api:replace-begin
		t.Run("ローカル環境ではuser_rolesベースAuthorizerが提供されINFOが出る", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)
			roleRepo := mock_user.NewMockRoleRepository(gomock.NewController(t))

			authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvLocal), Logger: logger, RoleRepo: roleRepo})
			require.NoError(t, err)
			assert.Equal(t, userrole.New(roleRepo), authorizer)
			// サンプル在時は local も実 authN と対で user_roles ベース authZ を配線し INFO を出す。
			assert.Len(t, logs.FilterMessage("user_roles-based authorizer wired").All(), 1)
		})
		// sample-api:replace-with
		// = t.Run("ローカル環境では全許可Authorizerが提供されWARNが出る", func(t *testing.T) {
		// = 	t.Parallel()
		// = 	logger, logs := logging.NewObservedTestLogger(t)
		// = 	authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvLocal), Logger: logger})
		// = 	require.NoError(t, err)
		// = 	expected, err := allowall.New(newAppCfg(t, config.EnvLocal))
		// = 	require.NoError(t, err)
		// = 	assert.Equal(t, expected, authorizer)
		// = 	assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		// = })
		// sample-api:replace-end

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

		// sample-api:begin
		t.Run("本番相当環境でRoleRepoが供給される場合、user_rolesベースAuthorizerが提供されINFOが出る", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)
			roleRepo := mock_user.NewMockRoleRepository(gomock.NewController(t))

			authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvProduction), Logger: logger, RoleRepo: roleRepo})
			require.NoError(t, err)
			assert.Equal(t, userrole.New(roleRepo), authorizer)
			assert.Len(t, logs.FilterMessage("user_roles-based authorizer wired").All(), 1)
		})
		// sample-api:end
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// local / ci / test 以外で、対応する認可実装が配線されていない環境は
		// すべて fail-closed（起動エラー）になること。
		failClosedEnvs := []string{
			// sample-api:replace-begin
			"unknown-env",
			// sample-api:replace-with
			// = config.EnvDevelopment,
			// = config.EnvStaging,
			// = config.EnvProduction,
			// = "unknown-env",
			// sample-api:replace-end
		}

		for _, env := range failClosedEnvs {
			t.Run(env+"では認可を配線せずエラーを返す", func(t *testing.T) {
				t.Parallel()

				logger := logging.NewTestLogger(t)

				authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, env), Logger: logger})
				require.Error(t, err)
				assert.Nil(t, authorizer)
			})
		}
	})
}

func Test_authzModule(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
