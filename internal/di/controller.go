package di

import (
	"boilerplate-go/internal/controller/handler/health"
	"boilerplate-go/internal/controller/handler/healthz"
	"boilerplate-go/internal/controller/handler/ready"
	"boilerplate-go/internal/controller/handler/v1/users"
	"boilerplate-go/internal/controller/handler/version"

	"go.uber.org/fx"
)

// ControllerModule は、コントローラー層の依存関係を提供するfx.Moduleです。
func ControllerModule() fx.Option {
	return fx.Module("controller",
		fx.Invoke(
			health.BindHandler,
			healthz.BindHandler,
			ready.BindHandler,
			version.BindHandler,
			// サンプルのハンドラー
			users.BindHandler,
		),
	)
}
