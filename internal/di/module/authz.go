package module

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/authz/allowall"
	"go-boilerplate/internal/logging"

	authzbd "go-boilerplate/internal/usecase/boundary/authz"

	"go.uber.org/fx"
)

// callerSkipCount は、ロギングラッパーが追加するフレーム数を補正するためのスキップ数です。
const callerSkipCount = 1

// authzModule は、認可（Authorizer）の依存を提供するfx.Moduleです。
// Authorizer は usecase 層から参照されるため、usecase に依存を供給する
// InfrastructureModule の一部として提供します。
func authzModule() fx.Option {
	return fx.Module("authz",
		fx.Provide(
			provideAuthorizer,
		),
	)
}

// provideAuthorizer は、現在の環境に対応した Authorizer を返します。現状の実装は全許可
// （allowall）の割り切りのみで、非本番環境（local / ci / test）以外での生成は allowall 自身が
// fail-closed で拒否します。ここではその結果に応じて、配線時の注意喚起（WARN）と拒否時の
// エラーログ（ERROR）を出すだけで、環境ごとの許否判断そのものは持ちません。
func provideAuthorizer(appCfg *config.ApplicationConfig, logger logging.Logger) (authzbd.Authorizer, error) {
	authorizer, err := allowall.New(appCfg)
	if err != nil {
		logger.Named("authz").CallerSkip(callerSkipCount).Error(
			context.Background(),
			"No authorizer configured for the current environment",
			logging.String("env", appCfg.Env()),
		)

		return nil, err
	}

	logger.Named("authz").CallerSkip(callerSkipCount).Warn(
		context.Background(),
		"Allow-all authorizer wired: every request is permitted (non-production only)",
		logging.String("env", appCfg.Env()),
	)

	return authorizer, nil
}
