package module

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/domain/user" // sample-api:line
	"go-boilerplate/internal/infrastructure/authz/allowall"

	// sample-api:replace-begin
	"go-boilerplate/internal/infrastructure/authz/userrole"
	// sample-api:replace-with
	// = "go-boilerplate/internal/infrastructure/authz/denyall"
	// sample-api:replace-end
	"go-boilerplate/internal/logging"

	authzbd "go-boilerplate/internal/usecase/boundary/authz"

	"go-boilerplate/pkg/xerrors"

	"go.uber.org/fx"
)

// callerSkipCount は、ロギングラッパーが追加するフレーム数を補正するためのスキップ数です。
const callerSkipCount = 1

// authorizerParams は、provideAuthorizer の依存を集約する fx パラメータです。
// RoleRepo はサンプル（user_roles）依存で、サンプル削除時にフィールドとプロバイダが対で除去されます。 // sample-api:line
type authorizerParams struct {
	fx.In

	AppCfg   *config.ApplicationConfig
	Logger   logging.Logger
	RoleRepo user.RoleRepository // sample-api:line
}

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

// provideAuthorizer は、環境に対応した Authorizer を返します。
// local / ci / test は全許可（allowall）の割り切り実装を配線します。
// dev / stg / prd（本番相当）は本番向けの認可実装を配線します
// （サンプルでは user_roles ベース、サンプル削除後は全拒否の deny-all 既定へ置換）。
// 未知の環境名は、誤った Authorizer を配線しないよう起動エラーにします。
func provideAuthorizer(p authorizerParams) (authzbd.Authorizer, error) {
	logger := p.Logger.Named("authz").CallerSkip(callerSkipCount)

	switch p.AppCfg.Env() {
	case config.EnvLocal, config.EnvCI, config.EnvTest:
		logger.Warn(
			context.Background(),
			"Allow-all authorizer wired: every request is permitted (non-production only)",
			logging.String("env", p.AppCfg.Env()),
		)

		return allowall.New(), nil
	case config.EnvDevelopment, config.EnvStaging, config.EnvProduction:
		// sample-api:replace-begin
		logger.Info(
			context.Background(),
			"user_roles-based authorizer wired",
			logging.String("env", p.AppCfg.Env()),
		)

		return userrole.New(p.RoleRepo), nil
		// sample-api:replace-with
		// = logger.Warn(
		// = context.Background(),
		// = "deny-all authorizer wired: every request is denied until an authorizer is opted in (safe default)",
		// = logging.String("env", p.AppCfg.Env()),
		// = )
		// =
		// = return denyall.New(), nil
		// sample-api:replace-end
	default:
		logger.Error(
			context.Background(),
			"No authorizer configured for the current environment",
			logging.String("env", p.AppCfg.Env()),
		)

		return nil, xerrors.New("unknown application environment: " + p.AppCfg.Env())
	}
}
