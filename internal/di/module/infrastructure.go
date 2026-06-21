package module

import (
	userqs "go-boilerplate/internal/infrastructure/rdb/query_service/user" // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/prefecture"     // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/user"           // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/system_query/healthcheck"
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
				),
			),
		),
		fx.Module("clock",
			fx.Provide(
				system.NewClock,
			),
		),
		fx.Module("security",
			fx.Provide(
				security.NewBcryptHasher,
			),
		),
	)
}
