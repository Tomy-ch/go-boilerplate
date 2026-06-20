package module

import (
	"go-boilerplate/internal/controller/handler/health"
	"go-boilerplate/internal/controller/handler/healthz"
	"go-boilerplate/internal/controller/handler/metrics"
	"go-boilerplate/internal/controller/handler/ready"
	"go-boilerplate/internal/controller/handler/v1/users"        // sample-api:line
	"go-boilerplate/internal/controller/handler/v1/users/detail" // sample-api:line
	"go-boilerplate/internal/controller/handler/v1/users/feed"   // sample-api:line
	"go-boilerplate/internal/controller/handler/v1/users/search" // sample-api:line
	"go-boilerplate/internal/controller/handler/version"

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
			metrics.BindHandler,
			// sample-api:begin
			// サンプルのハンドラー
			users.BindHandler,
			detail.BindHandler,
			feed.BindHandler,
			search.BindHandler,
			// sample-api:end
		),
	)
}
