package di

import (
	"boilerplate-go/internal/infrastructure/rdb/driver"

	"go.uber.org/fx"
)

// DatabaseModule は、データベース関連の依存関係を提供するfx.Moduleです。
func DatabaseModule() fx.Option {
	return fx.Module("db",
		fx.Provide(
			driver.NewDB,
			driver.NewTransactionManager,
			driver.NewLoggingDBProvider,
		),
	)
}
