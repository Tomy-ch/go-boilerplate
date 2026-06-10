package module

import (
	"go-boilerplate/internal/di/server/hook"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/driver/loggingdb"
	"go-boilerplate/internal/infrastructure/rdb/metrics"

	"go.uber.org/fx"
)

// DatabaseModule は、データベース関連の依存関係を提供するfx.Moduleです。
func DatabaseModule() fx.Option {
	return fx.Module("db",
		fx.Provide(
			driver.NewDB,
			driver.NewTransactionManager,
			loggingdb.NewLoggingDBProvider,
			metrics.NewRegisterer,
			metrics.New,
		),
		fx.Invoke(
			hook.RegisterDBCloseHooks,
			metrics.RegisterPoolStatsCollector,
		),
	)
}
