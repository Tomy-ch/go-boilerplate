package di

import (
	"boilerplate-go/internal/logging"

	"go.uber.org/fx"
)

// LoggingModule は、ロギング関連の依存関係を提供するfx.Moduleです。
func LoggingModule() fx.Option {
	return fx.Module("logging",
		fx.Provide(
			logging.New,
			logging.NewLogFields,
		),
	)
}
