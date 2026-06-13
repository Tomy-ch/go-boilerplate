package module

import (
	"go-boilerplate/internal/usecase/healthcheck"
	"go-boilerplate/internal/usecase/user"        // sample-api:line
	"go-boilerplate/internal/usecase/user/search" // sample-api:line

	"go.uber.org/fx"
)

// UsecaseModule は、ユースケース層の依存関係を提供するfx.Moduleです。
func UsecaseModule() fx.Option {
	return fx.Module("usecase",
		fx.Provide(
			healthcheck.New,
			// sample-api:begin
			// サンプルのユースケース
			user.New,
			search.New,
			// sample-api:end
		),
	)
}
