package module

import (
	userqs "boilerplate-go/internal/infrastructure/rdb/query_service/user"
	"boilerplate-go/internal/infrastructure/rdb/repository/prefecture"
	"boilerplate-go/internal/infrastructure/rdb/repository/user"
	"boilerplate-go/internal/infrastructure/rdb/system_query/healthcheck"
	"boilerplate-go/internal/infrastructure/security"
	"boilerplate-go/internal/infrastructure/system"

	"go.uber.org/fx"
)

// InfrastructureModule は、インフラストラクチャ層の依存関係を提供するfx.Moduleです。
func InfrastructureModule() fx.Option {
	return fx.Module("infrastructure",
		fx.Module("repository",
			fx.Provide(
				// サンプルのリポジトリ
				user.New,
				prefecture.New,
			),
		),
		fx.Module("query_service",
			fx.Provide(
				userqs.New,
			),
		),
		fx.Module("system_query",
			fx.Provide(
				healthcheck.New,
			),
		),
		fx.Module("system",
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
