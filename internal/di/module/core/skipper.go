package core

import (
	"go-boilerplate/internal/controller/httpstack/oapi/skipper"

	"go.uber.org/fx"
)

// SkipperModule は Oapi ミドルウェアのスキッパー機能を提供する Fx モジュールを返します。
func SkipperModule() fx.Option {
	return fx.Module("core.skipper",
		fx.Provide(
			skipper.New,
		),
	)
}
