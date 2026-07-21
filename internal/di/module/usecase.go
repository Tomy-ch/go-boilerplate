package module

import (
	"go-boilerplate/internal/observability"
	exchangerateuc "go-boilerplate/internal/usecase/exchangerate" // sample-api:line
	"go-boilerplate/internal/usecase/healthcheck"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/internal/usecase/outbox"
	prefectureuc "go-boilerplate/internal/usecase/prefecture"     // sample-api:line
	categoryuc "go-boilerplate/internal/usecase/product/category" // sample-api:line
	"go-boilerplate/internal/usecase/user"                        // sample-api:line
	"go-boilerplate/internal/usecase/user/search"                 // sample-api:line

	"go.uber.org/fx"
)

// UsecaseModule は、ユースケース層の依存関係を提供するfx.Moduleです。
func UsecaseModule() fx.Option {
	return fx.Module("usecase",
		fx.Provide(
			healthcheck.New,
			// 具象 IdempotencyMetrics を usecase 境界の Metrics / GCMetrics の双方として供給する。
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
			prefectureuc.New,
			categoryuc.New,
			// sample-api:end
		),
	)
}
