package module

import (
	userqs "go-boilerplate/internal/infrastructure/rdb/query_service/user" // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/prefecture"     // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/product"        // sample-api:line
	"go-boilerplate/internal/infrastructure/rdb/repository/user"           // sample-api:line
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
				product.New,
				// sample-api:end
			),
		),
		fx.Module("query_service",
			fx.Provide(
				// sample-api:begin
				// サンプルのクエリサービス
				userqs.New,
				// sample-api:end
			),
		),
		fx.Module("command_service",
			fx.Provide(
			// sample-api:begin
			// コマンドサービスは、このサンプルでは用意しませんが、必要に応じてここに追加します。
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
