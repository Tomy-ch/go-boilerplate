// Package module は、各レイヤの fx.Module（依存関係定義）を提供します。
package module

import (
	"go-boilerplate/internal/config"

	"go.uber.org/fx"
)

// ConfigModule は、アプリケーションの設定をDI用に提供するfx.Moduleです。
func ConfigModule() fx.Option {
	return fx.Module("config",
		fx.Provide(
			config.SetUpConfig,
		),
		fx.Provide(
			config.NewOperatingSystemConfig,
			config.NewApplicationConfig,
			config.NewServerConfig,
			config.NewDatabaseConfig,
			config.NewDBConnectionConfig,
			config.NewMetricsConfig,
			config.NewObservabilityConfig,
			config.NewSecurityConfig,
			config.NewSecureCookieConfig,
			config.NewWorkerConfig,
			config.NewOutboxConfig,
		),
		fx.Provide(
			config.NewTimeLocation,
		),
	)
}
