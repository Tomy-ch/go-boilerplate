package module

import (
	userqs "go-boilerplate/internal/infrastructure/rdb/query_service/user" // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/prefecture"
	"go-boilerplate/internal/infrastructure/rdb/repository/user" // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/system_query/healthcheck"
	"go-boilerplate/internal/infrastructure/security"
	"go-boilerplate/internal/infrastructure/system"

	"go.uber.org/fx"
)

// InfrastructureModule は、インフラストラクチャ層の依存関係を提供するfx.Moduleです。
func InfrastructureModule() fx.Option {
	return fx.Module("infrastructure",
		fx.Module("repository",
			fx.Provide(
				// sample-api:begin
				// サンプルのリポジトリ
				user.New,
				// sample-api:end
				prefecture.New,
			),
		),
		// sample-api:begin
		fx.Module("query_service",
			fx.Provide(
				userqs.New,
			),
		),
		// sample-api:end
		fx.Module("system_query",
			fx.Provide(
				healthcheck.New,
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
