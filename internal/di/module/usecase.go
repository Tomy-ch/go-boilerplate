package module

import (
	"boilerplate-go/internal/usecase/healthcheck"
	"boilerplate-go/internal/usecase/user"
	"boilerplate-go/internal/usecase/user/search"

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
