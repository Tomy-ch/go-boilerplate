package module

import (
	purchasecmd "go-boilerplate/internal/infrastructure/rdb/command_service/purchase"             // sample-api:line
	dashboardqs "go-boilerplate/internal/infrastructure/rdb/query_service/dashboard"              // sample-api:line
	productrankingqs "go-boilerplate/internal/infrastructure/rdb/query_service/product/ranking"   // sample-api:line
	purchasedetailqs "go-boilerplate/internal/infrastructure/rdb/query_service/purchase"          // sample-api:line
	purchasefeedqs "go-boilerplate/internal/infrastructure/rdb/query_service/purchase/feed"       // sample-api:line
	purchasesummaryqs "go-boilerplate/internal/infrastructure/rdb/query_service/purchase/summary" // sample-api:line
	cartrepo "go-boilerplate/internal/infrastructure/rdb/repository/cart"                         // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/prefecture"                            // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/product"                               // sample-api:line
	productcategory "go-boilerplate/internal/infrastructure/rdb/repository/product_category"      // sample-api:line
	productstatusrepo "go-boilerplate/internal/infrastructure/rdb/repository/productstatus"       // sample-api:line
	purchaserepo "go-boilerplate/internal/infrastructure/rdb/repository/purchase"                 // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/user"                                  // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/system_cqrs/healthcheck"
	idempotencysq "go-boilerplate/internal/infrastructure/rdb/system_cqrs/idempotency"
	outboxsq "go-boilerplate/internal/infrastructure/rdb/system_cqrs/outbox"

	"go.uber.org/fx"
)

// persistenceModule は、RDB 背後の永続化系依存（repository / query_service / command_service /
// system_cqrs）を提供する fx.Module です。詳細は internal/di/module/README.md § Design Policy を参照。
func persistenceModule() fx.Option {
	return fx.Module("persistence",
		fx.Module("repository",
			fx.Provide(
				// sample-api:begin
				user.New,
				user.NewRoleRepository,
				user.NewLockRepository,
				prefecture.New,
				productstatusrepo.New,
				productcategory.New,
				product.New,
				purchaserepo.New,
				cartrepo.New,
				// sample-api:end
			),
		),
		fx.Module("query_service",
			fx.Provide(
				// sample-api:begin
				// productrankingqs: 商品売上ランキング（docs/spec/product-ranking/usecase.md § Overview）
				productrankingqs.New,
				// purchasedetailqs / purchasefeedqs: 集約跨ぎ read 投影（docs/spec/purchase/usecase.md § GET 詳細 / GET 一覧）
				purchasedetailqs.New,
				purchasefeedqs.New,
				// purchasesummaryqs: 認証主体自身の購入集計（docs/spec/purchase/usecase.md § GET 集計）
				purchasesummaryqs.New,
				// dashboardqs: 購入・商品横断の admin 集計（docs/spec/dashboard/usecase.md § Overview）
				dashboardqs.New,
				// sample-api:end
			),
		),
		fx.Module("command_service",
			fx.Provide(
				// sample-api:begin
				// purchasecmd: 購入の原子的書き込み（docs/spec/purchase/usecase.md 冒頭ノート）
				purchasecmd.New,
				// sample-api:end
			),
		),
		fx.Module("system_cqrs",
			fx.Provide(
				healthcheck.New,
				idempotencysq.New,
				outboxsq.New,
			),
		),
	)
}
