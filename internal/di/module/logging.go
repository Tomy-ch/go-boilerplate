package module

import (
	"fmt"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

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
// extract は ctx から trace_id / span_id を抽出して注入する TraceExtractor です（nil のとき trace 注入なし）。
func provideLogger(
	appCfg *config.ApplicationConfig,
	logCore logging.LogCore,
	extract logging.TraceExtractor,
) (logging.Logger, error) {
	level, err := logging.ParseLevel(appCfg.LogLevel())
	if err != nil {
		return nil, xerrors.Wrap(err, fmt.Sprintf("invalid APP_LOG_LEVEL %q", appCfg.LogLevel()))
	}

	switch {
	case appCfg.IsProductionMode():
		return logging.WithCore(logging.NewJSONLogger(level, logging.LevelError(), extract), logCore), nil
	case appCfg.IsDevelopmentMode():
		return logging.WithCore(logging.NewConsoleLogger(level, logging.LevelWarn(), extract), logCore), nil
	default:
		return nil, xerrors.New("unknown app mode: " + appCfg.Mode())
	}
}
