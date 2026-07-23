package module

import (
	purchasecmd "go-boilerplate/internal/infrastructure/rdb/command_service/purchase"        // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/prefecture"                       // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/product"                          // sample-api:line
	productcategory "go-boilerplate/internal/infrastructure/rdb/repository/product_category" // sample-api:line
	productstatusrepo "go-boilerplate/internal/infrastructure/rdb/repository/productstatus"  // sample-api:line
	purchaserepo "go-boilerplate/internal/infrastructure/rdb/repository/purchase"            // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/user"                             // sample-api:line
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
			// クエリサービスは、このサンプルでは用意しませんが、必要に応じてここに追加します。
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
