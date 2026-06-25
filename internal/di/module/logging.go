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
// logCore が非 nil のときは、その core を Tee した Logger を返します。
func provideLogger(appCfg *config.ApplicationConfig, logCore logging.LogCore) (logging.Logger, error) {
	level, err := logging.ParseLevel(appCfg.LogLevel())
	if err != nil {
		return nil, fmt.Errorf("invalid APP_LOG_LEVEL %q: %w", appCfg.LogLevel(), err)
	}

	switch {
	case appCfg.IsProductionMode():
		return logging.WithCore(logging.NewJSONLogger(level, logging.LevelError()), logCore), nil
	case appCfg.IsDevelopmentMode():
		return logging.WithCore(logging.NewConsoleLogger(level, logging.LevelWarn()), logCore), nil
	default:
		return nil, fmt.Errorf("unknown app mode: %s", appCfg.Mode())
	}
}
