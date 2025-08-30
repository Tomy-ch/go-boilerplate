package di

import (
	userrepo "boilerplate-go/internal/infrastructure/rdb/repository/user"

	"go.uber.org/fx"
)

// RepositoryModule は、リポジトリ層の依存関係を提供するfx.Moduleです。
func RepositoryModule() fx.Option {
	return fx.Module("repository",
		fx.Provide(
			userrepo.New,
		),
	)
}
