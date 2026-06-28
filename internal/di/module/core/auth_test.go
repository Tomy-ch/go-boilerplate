package core

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/auth/local"
	"go-boilerplate/internal/logging"
	authbd "go-boilerplate/internal/usecase/boundary/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestAuthModule(t *testing.T) {
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
			fx.Provide(logging.NewTestLogger),
			AuthnModule(),
			fx.Populate(&a),
		)

		require.NoError(t, app.Start(context.Background()))
		require.NotNil(t, a)
		require.NotPanics(t, func() { _ = a })
		require.NoError(t, app.Stop(context.Background()))
	})
}

func Test_provideAuthenticator(t *testing.T) {
	t.Parallel()

	t.Run("ローカル環境では local.Authenticator が提供されWARNが出る", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		appCfg.SetApplicationEnv(t, config.EnvLocal)
		logger, logs := logging.NewObservedTestLogger(t)
		authenticator, err := provideAuthenticator(appCfg, logger)
		require.NoError(t, err)

		la := local.New()
		assert.Equal(t, la, authenticator)
		// 開発用スタブ配線時に WARN で注意喚起されること。
		assert.Len(t, logs.FilterMessage("Local authenticator wired: authentication is stubbed (non-production only)").All(), 1)
	})

	t.Run("CI環境では local.Authenticator が提供される", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		appCfg.SetApplicationEnv(t, config.EnvCI)
		logger := logging.NewTestLogger(t)
		authenticator, err := provideAuthenticator(appCfg, logger)
		require.NoError(t, err)

		la := local.New()
		assert.Equal(t, la, authenticator)
	})

	t.Run("テスト環境では local.Authenticator が提供される", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		appCfg.SetApplicationEnv(t, config.EnvTest)
		logger := logging.NewTestLogger(t)
		authenticator, err := provideAuthenticator(appCfg, logger)
		require.NoError(t, err)

		la := local.New()
		assert.Equal(t, la, authenticator)
	})

	// local / ci / test 以外（本番相当）はすべて fail-closed でエラーを返すこと。
	envs := map[string]string{
		"development環境": config.EnvDevelopment,
		"staging環境":     config.EnvStaging,
		"production環境":  config.EnvProduction,
	}
	for name, env := range envs {
		t.Run(name+"では Authenticator を配線せずエラーを返す", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			appCfg.SetApplicationEnv(t, env)
			logger := logging.NewTestLogger(t)
			authenticator, err := provideAuthenticator(appCfg, logger)
			require.Error(t, err)
			require.Nil(t, authenticator)
		})
	}
}
