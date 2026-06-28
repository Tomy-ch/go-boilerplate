package module

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/authz/allowall"
	"go-boilerplate/internal/logging"

	authzbd "go-boilerplate/internal/usecase/boundary/authz"

	"go-boilerplate/pkg/xerrors"

	"go.uber.org/fx"
)

// callerSkipCount は、ロギングラッパーが追加するフレーム数を補正するためのスキップ数です。
const callerSkipCount = 1

// authzModule は、認可（Authorizer）の依存を提供するfx.Moduleです。
// Authorizer は usecase 層から参照されるため、usecase に依存を供給する
// InfrastructureModule の一部として提供します（authn の Authenticator が
// controller 側の core モジュールに置かれるのと対照的な配置です）。
func authzModule() fx.Option {
	return fx.Module("authz",
		fx.Provide(
			provideAuthorizer,
		),
	)
}

// provideAuthorizer は、環境（EnvLocal / EnvCI / EnvTest）に対応した Authorizer を返します。
// 現状の実装は全許可（allowall）の割り切りであるため、本番相当の環境では誤って全許可を
// 配線しないようエラーを返し、RBAC / 外部ポリシーエンジン実装への差し替えを強制します。
func provideAuthorizer(appCfg *config.ApplicationConfig, logger logging.Logger) (authzbd.Authorizer, error) {
	switch appCfg.Env() {
	case config.EnvLocal, config.EnvCI, config.EnvTest:
		return allowall.New(), nil
	default:
		logger.Named("authz").CallerSkip(callerSkipCount).Error(
			"No authorizer configured for the current environment",
			logging.String("env", appCfg.Env()),
		)

		return nil, xerrors.New("no authorizer configured for environment: " + appCfg.Env())
	}
}
