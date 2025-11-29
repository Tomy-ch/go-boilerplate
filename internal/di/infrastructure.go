package di

import (
	"boilerplate-go/internal/infrastructure/rdb/repository/user"
	"boilerplate-go/internal/infrastructure/rdb/system_query/healthcheck"

	"go.uber.org/fx"
)

// InfrastructureModule は、インフラストラクチャ層の依存関係を提供するfx.Moduleです。
func InfrastructureModule() fx.Option {
	return fx.Module("infrastructure",
		fx.Module("repository",
			fx.Provide(
				user.New,
			),
		),
		fx.Module("query_service",
			fx.Provide(),
		),
		fx.Module("system_query",
			fx.Provide(
				healthcheck.New,
			),
		),
	)
}
