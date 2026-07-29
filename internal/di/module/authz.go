package module

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/domain/user" // sample-api:line
	"go-boilerplate/internal/infrastructure/authz/allowall"
	"go-boilerplate/internal/infrastructure/authz/userrole" // sample-api:line
	"go-boilerplate/internal/logging"

	authzbd "go-boilerplate/internal/usecase/boundary/authz"

	"go-boilerplate/pkg/xerrors"

	"go.uber.org/fx"
)

// callerSkipCount は、ロギングラッパーが追加するフレーム数を補正するためのスキップ数です。
const callerSkipCount = 1

// errNoAuthorizerForEnv は、現在の環境に対応する Authorizer が無く配線に失敗した場合のエラーです。
var errNoAuthorizerForEnv = xerrors.New("no authorizer configured for environment")

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

// provideAuthorizer は、環境ごとに対応する Authorizer を選んで返します。
// 環境ごとに switch の case で個別に配線でき、対応する case が無い環境は誤った Authorizer を配線しないよう default で起動エラーにします（fail-closed）。
func provideAuthorizer(p authorizerParams) (authzbd.Authorizer, error) {
	logger := p.Logger.Named("authz").CallerSkip(callerSkipCount)

	switch p.AppCfg.Env() {
	// sample-api:replace-begin
	case config.EnvCI, config.EnvTest:
		// sample-api:replace-with
		// = case config.EnvLocal, config.EnvCI, config.EnvTest:
		// sample-api:replace-end
		authorizer, err := allowall.New(p.AppCfg)
		if err != nil {
			logger.Error(
				context.Background(),
				"No authorizer configured for the current environment",
				logging.String("env", p.AppCfg.Env()),
			)

			return nil, err
		}

		logger.Warn(
			context.Background(),
			"Allow-all authorizer wired: every request is permitted (non-production only)",
			logging.String("env", p.AppCfg.Env()),
		)

		return authorizer, nil
	// sample-api:begin
	case config.EnvLocal, config.EnvDevelopment, config.EnvStaging, config.EnvProduction:
		logger.Info(
			context.Background(),
			"user_roles-based authorizer wired",
			logging.String("env", p.AppCfg.Env()),
		)

		return userrole.New(p.RoleRepo), nil
	// sample-api:end
	default:
		logger.Error(
			context.Background(),
			"No authorizer configured for the current environment",
			logging.String("env", p.AppCfg.Env()),
		)

		return nil, xerrors.Wrap(errNoAuthorizerForEnv, p.AppCfg.Env())
	}
}
