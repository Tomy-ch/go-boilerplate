package di

import (
	"boilerplate-go/internal/config"

	"go.uber.org/fx"
)

// ConfigModule は、アプリケーションの設定をDI用に提供するfx.Moduleです。
func ConfigModule() fx.Option {
	return fx.Module("config",
		fx.Provide(
			config.SetUpConfig,
		),
		fx.Provide(
			config.NewOSConfig,
			config.NewApplicationConfig,
			config.NewDatabaseConfig,
			config.NewDBConnectionConfig,
			config.NewMetricsConfig,
			config.NewObservabilityConfig,
			config.NewSecurityConfig,
			config.NewServerConfig,
		),
	)
}
