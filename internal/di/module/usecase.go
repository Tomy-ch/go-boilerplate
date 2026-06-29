package module

import (
	"go-boilerplate/internal/observability"
	exchangerateuc "go-boilerplate/internal/usecase/exchangerate" // sample-api:line
	"go-boilerplate/internal/usecase/healthcheck"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/internal/usecase/user"        // sample-api:line
	"go-boilerplate/internal/usecase/user/search" // sample-api:line

	"go.uber.org/fx"
)

// UsecaseModule は、ユースケース層の依存関係を提供するfx.Moduleです。
func UsecaseModule() fx.Option {
	return fx.Module("usecase",
		fx.Provide(
			healthcheck.New,
			// 具象 IdempotencyMetrics を usecase 境界の Metrics / GCMetrics の双方として供給する。
			// fx.As は単一結果を各 interface へ写像するため、interface ごとに annotation を分ける。
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
			// sample-api:begin
			// サンプルのユースケース
			user.New,
			search.New,
			exchangerateuc.New,
			// sample-api:end
		),
	)
}
