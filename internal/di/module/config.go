// Package module は、DIでの設定関連の依存関係を提供します。
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
			config.NewOperationSystemConfig,
			config.NewApplicationConfig,
			config.NewServerConfig,
			config.NewDatabaseConfig,
			config.NewDBConnectionConfig,
			config.NewMetricsConfig,
			config.NewObservabilityConfig,
			config.NewSecurityConfig,
			config.NewSecureCookieConfig,
			config.NewAuthConfig,
			config.NewIPRateLimitConfig,
		),
		fx.Provide(
			config.NewTimeLocation,
		),
	)
}
