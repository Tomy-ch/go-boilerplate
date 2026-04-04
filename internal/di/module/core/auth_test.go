package core

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/auth/local"
	"go-boilerplate/internal/logging"
	authbd "go-boilerplate/internal/usecase/boundary/auth"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestAuthModule(t *testing.T) {
	t.Run("fx アプリで Authenticator が提供される", func(t *testing.T) {
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
	t.Run("ローカル環境では local.Authenticator が提供される", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		appCfg.SetApplicationEnv(t, config.EnvLocal)
		logger := logging.NewTestLogger(t)

		authenticator := provideAuthenticator(appCfg, logger)

		la := local.New()
		require.Equal(t, la, authenticator)
	})

	t.Run("CI環境では local.Authenticator が提供される", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		appCfg.SetApplicationEnv(t, config.EnvCI)
		logger := logging.NewTestLogger(t)

		authenticator := provideAuthenticator(appCfg, logger)

		la := local.New()
		require.Equal(t, la, authenticator)
	})

	t.Run("テスト環境では local.Authenticator が提供される", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		appCfg.SetApplicationEnv(t, config.EnvTest)
		logger := logging.NewTestLogger(t)

		authenticator := provideAuthenticator(appCfg, logger)

		la := local.New()
		require.Equal(t, la, authenticator)
	})

	t.Run("その他の環境では本番用 Authenticator が提供される", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		appCfg.SetApplicationEnv(t, config.EnvProduction)
		logger := logging.NewTestLogger(t)

		authenticator := provideAuthenticator(appCfg, logger)
		// 本番用 Authenticator は未実装のため nil になる想定
		require.Nil(t, authenticator)
	})
}
