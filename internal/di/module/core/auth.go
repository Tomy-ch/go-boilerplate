package core

import (
	"fmt"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/oapi/auth"
	"go-boilerplate/internal/infrastructure/auth/local"

	authbd "go-boilerplate/internal/usecase/boundary/auth"

	"go.uber.org/fx"
)

// AuthnModule は、認証関連の依存関係を提供するfxモジュールを返します。
func AuthnModule() fx.Option {
	return fx.Module("core.authn",
		fx.Provide(
			provideAuthenticator,
			auth.NewAuthenticator,
		),
	)
}

// provideAuthenticator は、Authenticator のコンストラクタを提供します。
func provideAuthenticator(appCfg *config.ApplicationConfig) (authbd.Authenticator, error) {
	switch appCfg.Env() {
	case config.EnvLocal, config.EnvCI, config.EnvTest:
		return local.New(), nil
	default:
		return nil, fmt.Errorf("no authenticator configured for environment: %s", appCfg.Env())
	}
}
