package module

import (
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
			fx.Provide(),
		),
		fx.Module("query_service",
			fx.Provide(),
		),
		fx.Module("command_service",
			fx.Provide(),
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
