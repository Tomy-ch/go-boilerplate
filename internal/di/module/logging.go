package module

import (
	"fmt"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"

	"go.uber.org/fx"
)

// LoggingModule は、ロギング関連の依存関係を提供するfx.Moduleです。
func LoggingModule() fx.Option {
	return fx.Module("logging",
		fx.Provide(
			provideLogger,
			logging.NewLogFields,
		),
	)
}

// provideLogger は、アプリケーション設定に応じた Logger を生成します。
func provideLogger(appCfg *config.ApplicationConfig) (logging.Logger, error) {
	level, err := logging.ParseLevel(appCfg.LogLevel())
	if err != nil {
		return nil, fmt.Errorf("invalid APP_LOG_LEVEL %q: %w", appCfg.LogLevel(), err)
	}

	switch {
	case appCfg.IsProductionMode():
		return logging.NewJSONLogger(level, logging.LevelError()), nil
	case appCfg.IsDevelopmentMode():
		return logging.NewConsoleLogger(level, logging.LevelWarn()), nil
	default:
		return nil, fmt.Errorf("unknown app mode: %s", appCfg.Mode())
	}
}
