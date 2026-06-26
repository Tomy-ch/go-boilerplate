package module

import (
	"go-boilerplate/internal/infrastructure/system"

	"go.uber.org/fx"
)

// clockModule は、時刻・待機関連の依存を提供するfx.Moduleです。
// SystemModule() の "system" ラベルとの衝突を避けるため "clock" としています。
func clockModule() fx.Option {
	return fx.Module("clock",
		fx.Provide(
			system.NewClock,
			system.NewSleeper,
		),
	)
}
