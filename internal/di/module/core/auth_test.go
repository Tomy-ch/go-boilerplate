package core

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/auth/local"
	"go-boilerplate/internal/infrastructure/httpclient"
	mock_httpclient "go-boilerplate/internal/infrastructure/httpclient/mock"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
	authbd "go-boilerplate/internal/usecase/boundary/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/mock/gomock"
)

// newAuthParams は、指定した環境で provideAuthenticator を呼ぶための依存を組み立てます。
func newAuthParams(t *testing.T, env string, logger logging.Logger) authenticatorParams {
	t.Helper()
	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	appCfg.SetApplicationEnv(t, env)

	return authenticatorParams{
		AppCfg:  appCfg,
		AuthCfg: config.NewAuthConfig(cfg),
		Clock:   system.NewClock(),
		Logger:  logger,
	}
}

func TestAuthnModule(t *testing.T) {
	t.Parallel()

	t.Run("fx アプリで Authenticator が提供される", func(t *testing.T) {
		t.Parallel()

		var a authbd.Authenticator
		app := fx.New(
			fx.Provide(func() testing.TB { return t }),
			fx.Provide(func() *testing.T { return t }),
			fx.Provide(config.MockConfigForTest),
			fx.Provide(config.NewApplicationConfig),
			fx.Provide(config.NewAuthConfig),
			fx.Provide(system.NewClock),
			fx.Provide(logging.NewTestLogger),
			// HTTPClient は infra 層が常設提供する必須依存。test 環境ではスタブ認証で未使用だが、
			// authenticatorParams の解決に必要なためモックを供給する。
			fx.Provide(func() httpclient.Client {
				return mock_httpclient.NewMockClient(gomock.NewController(t))
			}),
			AuthnModule(),
			fx.Populate(&a),
		)

		require.NoError(t, app.Start(context.Background()))
		t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
		assert.NotNil(t, a)
	})
}

func Test_provideAuthenticator(t *testing.T) {
	t.Parallel()

	t.Run("ローカル環境ではJWKS authenticatorを配線しAUTH未設定なら fail-closed になる", func(t *testing.T) {
		t.Parallel()

		logger := logging.NewTestLogger(t)
		authenticator, err := provideAuthenticator(newAuthParams(t, config.EnvLocal, logger))
		require.Error(t, err)
		assert.Nil(t, authenticator)
	})

	t.Run("CI環境では local.Authenticator が提供されWARNが出る", func(t *testing.T) {
		t.Parallel()

		logger, logs := logging.NewObservedTestLogger(t)
		authenticator, err := provideAuthenticator(newAuthParams(t, config.EnvCI, logger))
		require.NoError(t, err)

		la := local.New()
		assert.Equal(t, la, authenticator)
		// スタブ配線時に WARN で注意喚起されること。
		assert.Len(t, logs.FilterMessage("Local authenticator wired: authentication is stubbed (non-production only)").All(), 1)
	})

	t.Run("テスト環境では local.Authenticator が提供される", func(t *testing.T) {
		t.Parallel()

		logger := logging.NewTestLogger(t)
		authenticator, err := provideAuthenticator(newAuthParams(t, config.EnvTest, logger))
		require.NoError(t, err)

		la := local.New()
		assert.Equal(t, la, authenticator)
	})

	t.Run("development環境でAUTH設定が無い場合はfail-closedでエラーを返す", func(t *testing.T) {
		t.Parallel()

		logger := logging.NewTestLogger(t)
		authenticator, err := provideAuthenticator(newAuthParams(t, config.EnvDevelopment, logger))
		require.Error(t, err)
		assert.Nil(t, authenticator)
	})

	// staging / production はまだ実 IdP 未配線のため、default で fail-closed になること。
	assertFailClosed := func(t *testing.T, env string) {
		t.Helper()
		logger := logging.NewTestLogger(t)
		authenticator, err := provideAuthenticator(newAuthParams(t, env, logger))
		require.Error(t, err)
		assert.Nil(t, authenticator)
	}

	t.Run("staging環境では Authenticator を配線せずエラーを返す", func(t *testing.T) {
		t.Parallel()
		assertFailClosed(t, config.EnvStaging)
	})

	t.Run("production環境では Authenticator を配線せずエラーを返す", func(t *testing.T) {
		t.Parallel()
		assertFailClosed(t, config.EnvProduction)
	})
}
