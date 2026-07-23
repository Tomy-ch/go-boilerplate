package module

import (
	"go-boilerplate/internal/controller/handler/health"
	"go-boilerplate/internal/controller/handler/healthz"
	"go-boilerplate/internal/controller/handler/metrics"
	"go-boilerplate/internal/controller/handler/ready"
	addresseshandler "go-boilerplate/internal/controller/handler/v1/addresses"                  // sample-api:line
	exchangeratehandler "go-boilerplate/internal/controller/handler/v1/exchangerate"            // sample-api:line
	prefectureshandler "go-boilerplate/internal/controller/handler/v1/prefectures"              // sample-api:line
	productcategorieshandler "go-boilerplate/internal/controller/handler/v1/product-categories" // sample-api:line
	productstatuseshandler "go-boilerplate/internal/controller/handler/v1/product-statuses"     // sample-api:line
	productshandler "go-boilerplate/internal/controller/handler/v1/products"                    // sample-api:line
	productsdetailhandler "go-boilerplate/internal/controller/handler/v1/products/detail"       // sample-api:line
	"go-boilerplate/internal/controller/handler/v1/users"                                       // sample-api:line
	"go-boilerplate/internal/controller/handler/v1/users/detail"                                // sample-api:line
	"go-boilerplate/internal/controller/handler/v1/users/feed"                                  // sample-api:line
	"go-boilerplate/internal/controller/handler/v1/users/search"                                // sample-api:line
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
			exchangeratehandler.BindHandler,
			addresseshandler.BindHandler,
			prefectureshandler.BindHandler,
			productstatuseshandler.BindHandler,
			productcategorieshandler.BindHandler,
			productshandler.BindHandler,
			productsdetailhandler.BindHandler,
			// sample-api:end
		),
	)
}
