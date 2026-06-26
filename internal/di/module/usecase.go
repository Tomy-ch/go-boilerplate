package module

import (
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
