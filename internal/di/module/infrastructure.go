package module

import (
	userqs "go-boilerplate/internal/infrastructure/rdb/query_service/user"
	"go-boilerplate/internal/infrastructure/rdb/repository/prefecture"
	"go-boilerplate/internal/infrastructure/rdb/repository/user"
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
