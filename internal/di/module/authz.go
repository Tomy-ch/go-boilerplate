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

// authorizerParams は、provideAuthorizer の依存を集約する fx パラメータです。
// RoleRepo はサンプル（user_roles）依存で、optional にすることで削除後も Authorizer 配線が壊れないようにしています。 // sample-api:line
type authorizerParams struct {
	fx.In

	AppCfg   *config.ApplicationConfig
	Logger   logging.Logger
	RoleRepo user.RoleRepository `optional:"true"` // sample-api:line
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
// EnvLocal / EnvCI / EnvTest は全許可（allowall）の割り切り実装を配線します。
// default（本番相当）環境はサンプルの user_roles ベース実装（userrole）を配線します。 // sample-api:line
// user_roles 実装が供給されない場合は、誤って全許可を配線しないようエラーを返し、実装差し替えを強制します。
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
	default:
		// sample-api:begin
		if p.RoleRepo != nil {
			logger.Info(
				context.Background(),
				"user_roles-based authorizer wired",
				logging.String("env", p.AppCfg.Env()),
			)

			return userrole.New(p.RoleRepo), nil
		}
		// sample-api:end
		logger.Error(
			context.Background(),
			"No authorizer configured for the current environment",
			logging.String("env", p.AppCfg.Env()),
		)

		return nil, xerrors.New("no authorizer configured for environment: " + p.AppCfg.Env())
	}
}
