package module

import (
	"go-boilerplate/internal/system"

	"go.uber.org/fx"
)

// SystemModule は、システム情報関連の依存関係を提供するfx.Moduleです。
func SystemModule() fx.Option {
	return fx.Module("system",
		fx.Provide(
			system.NewBuildInfo,
		),
	)
}
