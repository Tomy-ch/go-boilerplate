package module

import (
	"boilerplate-go/internal/di/server/hook"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/driver/loggingdriver"
	"boilerplate-go/internal/infrastructure/rdb/metrics"

	"go.uber.org/fx"
)

// DatabaseModule は、データベース関連の依存関係を提供するfx.Moduleです。
func DatabaseModule() fx.Option {
	return fx.Module("db",
		fx.Provide(
			driver.NewDB,
			driver.NewTransactionManager,
			loggingdriver.NewLoggingDBProvider,
			metrics.New,
		),
		fx.Invoke(
			hook.RegisterDBCloseHooks,
			metrics.RegisterPoolStatsCollector,
		),
	)
}
