package module

import (
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/healthcheck"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/internal/usecase/outbox"

	"go.uber.org/fx"
)

// UsecaseModule は、ユースケース層の依存関係を提供するfx.Moduleです。
func UsecaseModule() fx.Option {
	return fx.Module("usecase",
		fx.Provide(
			healthcheck.New,
			fx.Annotate(
				observability.NewIdempotencyMetrics,
				fx.As(new(idempotency.Metrics)),
				fx.As(new(idempotency.GCMetrics)),
			),
			idempotency.NewDeps,
			idempotency.NewGC,
			outbox.NewEmit,
			outbox.NewGC,
			outbox.NewReplay,
		),
	)
}
