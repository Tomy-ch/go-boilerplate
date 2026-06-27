package core

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/oapi/auth"
	"go-boilerplate/internal/infrastructure/auth/local"
	"go-boilerplate/internal/logging"

	authbd "go-boilerplate/internal/usecase/boundary/auth"

	"go-boilerplate/pkg/xerrors"

	"go.uber.org/fx"
)

// callerSkipCount は、ロギングラッパーが追加するフレーム数を補正するためのスキップ数です。
const callerSkipCount = 1

// AuthnModule は、認証関連の依存関係を提供するfxモジュールを返します。
func AuthnModule() fx.Option {
	return fx.Module("core.authn",
		fx.Provide(
			provideAuthenticator,
			auth.NewAuthenticator,
		),
	)
}

// provideAuthenticator は、環境（EnvLocal / EnvCI / EnvTest）に対応した Authenticator を返します。
// それ以外の環境ではエラーを返し、FX の起動に失敗します。
func provideAuthenticator(appCfg *config.ApplicationConfig, logger logging.Logger) (authbd.Authenticator, error) {
	switch appCfg.Env() {
	case config.EnvLocal, config.EnvCI, config.EnvTest:
		return local.New(), nil
	default:
		logger.Named("core.authn").CallerSkip(callerSkipCount).Error(
			"No authenticator configured for the current environment",
			logging.String("env", appCfg.Env()),
		)

		return nil, xerrors.New("no authenticator configured for environment: " + appCfg.Env())
	}
}
