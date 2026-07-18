package module

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/domain/user"                // sample-api:line
	mock_user "go-boilerplate/internal/domain/user/mock" // sample-api:line
	"go-boilerplate/internal/infrastructure/authz/allowall"

	// sample-api:replace-begin
	"go-boilerplate/internal/infrastructure/authz/userrole"
	// sample-api:replace-with
	// = "go-boilerplate/internal/infrastructure/authz/denyall"
	// sample-api:replace-end
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

		t.Run("ローカル環境では全許可Authorizerが提供されWARNが出る", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)

			authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvLocal), Logger: logger})
			require.NoError(t, err)
			assert.Equal(t, allowall.New(), authorizer)
			assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		})

		t.Run("CI環境では全許可Authorizerが提供されWARNが出る", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)

			authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvCI), Logger: logger})
			require.NoError(t, err)
			assert.Equal(t, allowall.New(), authorizer)
			assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		})

		t.Run("テスト環境では全許可Authorizerが提供されWARNが出る", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)

			authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvTest), Logger: logger})
			require.NoError(t, err)
			assert.Equal(t, allowall.New(), authorizer)
			assert.Len(t, logs.FilterMessage("Allow-all authorizer wired: every request is permitted (non-production only)").All(), 1)
		})

		// sample-api:replace-begin
		t.Run("本番相当環境でRoleRepoが供給される場合、user_rolesベースAuthorizerが提供されINFOが出る", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)
			roleRepo := mock_user.NewMockRoleRepository(gomock.NewController(t))

			authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvProduction), Logger: logger, RoleRepo: roleRepo})
			require.NoError(t, err)
			assert.Equal(t, userrole.New(roleRepo), authorizer)
			assert.Len(t, logs.FilterMessage("user_roles-based authorizer wired").All(), 1)
		})
		// sample-api:replace-with
		// = t.Run("本番相当環境ではdeny-all既定のAuthorizerが提供されWARNが出る", func(t *testing.T) {
		// = t.Parallel()
		// =
		// = logger, logs := logging.NewObservedTestLogger(t)
		// =
		// = authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, config.EnvProduction), Logger: logger})
		// = require.NoError(t, err)
		// = assert.Equal(t, denyall.New(), authorizer)
		// = assert.Len(t, logs.FilterMessage("deny-all authorizer wired: every request is denied until an authorizer is opted in (safe default)").All(), 1)
		// = })
		// sample-api:replace-end
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知の環境名では全許可を配線せずエラーを返す", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)

			authorizer, err := provideAuthorizer(authorizerParams{AppCfg: newAppCfg(t, "unknown-env"), Logger: logger})
			require.Error(t, err)
			assert.Nil(t, authorizer)
		})
	})
}
