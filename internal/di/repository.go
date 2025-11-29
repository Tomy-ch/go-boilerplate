package di

import (
	userrepo "boilerplate-go/internal/infrastructure/rdb/repository/user"
	healthcheck "boilerplate-go/internal/infrastructure/rdb/system_query/health_check"

	"go.uber.org/fx"
)

// RepositoryModule は、リポジトリ層の依存関係を提供するfx.Moduleです。
func RepositoryModule() fx.Option {
	return fx.Module("repository",
		fx.Provide(
			healthcheck.New,
			userrepo.New,
		),
	)
}
