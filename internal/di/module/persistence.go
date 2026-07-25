package module

import (
	purchasecmd "go-boilerplate/internal/infrastructure/rdb/command_service/purchase"             // sample-api:line
	productrankingqs "go-boilerplate/internal/infrastructure/rdb/query_service/product/ranking"   // sample-api:line
	purchasedetailqs "go-boilerplate/internal/infrastructure/rdb/query_service/purchase"          // sample-api:line
	purchasesummaryqs "go-boilerplate/internal/infrastructure/rdb/query_service/purchase/summary" // sample-api:line
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

// persistenceModule は、RDB を背後に持つ永続化系の依存（repository / query_service /
// command_service / system_cqrs）を提供するfx.Moduleです。DatabaseModule() の db 接続層
// とは区別され、その上に載るデータアクセス実装をまとめます。
func persistenceModule() fx.Option {
	return fx.Module("persistence",
		fx.Module("repository",
			fx.Provide(
				// sample-api:begin
				// サンプルのリポジトリ
				user.New,
				user.NewRoleRepository,
				prefecture.New,
				productstatusrepo.New,
				productcategory.New,
				product.New,
				purchaserepo.New,
				// sample-api:end
			),
		),
		fx.Module("query_service",
			fx.Provide(
				// sample-api:begin
				// サンプルのクエリサービス（購入明細を集計した商品売上ランキング）
				productrankingqs.New,
				// サンプルのクエリサービス（購入詳細の集約跨ぎ read 投影）
				purchasedetailqs.New,
				// サンプルのクエリサービス（認証主体自身の購入集計）
				purchasesummaryqs.New,
				// sample-api:end
			),
		),
		fx.Module("command_service",
			fx.Provide(
				// sample-api:begin
				// サンプルのコマンドサービス（購入の原子的書き込み）
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
