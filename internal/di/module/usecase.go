package module

import (
	"go-boilerplate/internal/usecase/healthcheck"
	"go-boilerplate/internal/usecase/user"
	"go-boilerplate/internal/usecase/user/search"

	"go.uber.org/fx"
)

// UsecaseModule は、ユースケース層の依存関係を提供するfx.Moduleです。
func UsecaseModule() fx.Option {
	return fx.Module("usecase",
		fx.Provide(
			healthcheck.New,
			// サンプルのユースケース
			user.New,
			search.New,
		),
	)
}
