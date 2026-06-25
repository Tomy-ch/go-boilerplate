package module

import (
	exchangerateext "go-boilerplate/internal/infrastructure/external/exchangerate" // sample-api:line
	"go-boilerplate/internal/infrastructure/httpclient"
	userqs "go-boilerplate/internal/infrastructure/rdb/query_service/user" // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/prefecture"     // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/user"           // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/system_query/healthcheck"
	idempotencysq "go-boilerplate/internal/infrastructure/rdb/system_query/idempotency"
	"go-boilerplate/internal/infrastructure/security"
	"go-boilerplate/internal/infrastructure/system"

	"go.uber.org/fx"
)

// InfrastructureModule は、インフラストラクチャ層の依存関係を提供するfx.Moduleです。
func InfrastructureModule() fx.Option {
	return fx.Module("infrastructure",
		fx.Module("persistence",
			fx.Module("repository",
				fx.Provide(
					// sample-api:begin
					// サンプルのリポジトリ
					user.New,
					prefecture.New,
					// sample-api:end
				),
			),
			fx.Module("query_service",
				fx.Provide(
					// sample-api:begin
					// サンプルのクエリサービス
					userqs.New,
					// sample-api:end
				),
			),
			fx.Module("command_service",
				fx.Provide(
				// sample-api:begin
				// コマンドサービスは、このサンプルでは用意しませんが、必要に応じてここに追加します。
				// sample-api:end
				),
			),
			fx.Module("system_query",
				fx.Provide(
					healthcheck.New,
					idempotencysq.New,
				),
			),
		),
		clockModule(),
		httpClientModule(),
		externalModule(),
		fx.Module("security",
			fx.Provide(
				security.NewBcryptHasher,
			),
		),
	)
}

// clockModule は、時刻・待機関連の依存を提供するfx.Moduleです。
func clockModule() fx.Option {
	return fx.Module("clock",
		fx.Provide(
			system.NewClock,
			system.NewSleeper,
		),
	)
}

// httpClientModule は、resilient な外部 HTTP client substrate を提供するfx.Moduleです。
func httpClientModule() fx.Option {
	return fx.Module("httpclient",
		fx.Provide(
			fx.Annotate(
				httpclient.NewRegistryFromProfiles,
				fx.ParamTags(`group:"httpclient_profiles"`),
			),
			httpclient.New,
		),
	)
}

// externalModule は、外部サービス gateway を提供するfx.Moduleです。
func externalModule() fx.Option {
	return fx.Module("external",
		fx.Provide(
			// sample-api:begin
			// サンプルの外部サービス gateway（DTO モード）
			exchangerateext.NewEndpoint,
			exchangerateext.New,
			fx.Annotate(
				exchangerateext.NewDownstreamProfile,
				fx.ResultTags(`group:"httpclient_profiles"`),
			),
			// sample-api:end
		),
	)
}
