package di

import (
	healthcheckuc "boilerplate-go/internal/usecase/healthcheck"
	useruc "boilerplate-go/internal/usecase/user"

	"go.uber.org/fx"
)

// UsecaseModule は、ユースケース層の依存関係を提供するfx.Moduleです。
func UsecaseModule() fx.Option {
	return fx.Module("usecase",
		fx.Provide(
			useruc.New,
			healthcheckuc.New,
		),
	)
}
