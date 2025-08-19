package di

import (
	"boilerplate-go/internal/infrastructure/rdb"

	"go.uber.org/fx"
)

// DatabaseModule は、データベース関連の依存関係を提供するfx.Moduleです。
func DatabaseModule() fx.Option {
	return fx.Module("db",
		fx.Provide(
			rdb.NewDB,
			rdb.NewTransactionManager,
		),
	)
}
