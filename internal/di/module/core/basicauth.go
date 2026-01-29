package core

import (
	"boilerplate-go/internal/controller/httpstack/basicauth"

	"go.uber.org/fx"
)

// BasicAuthModule は Basic 認証機能を提供する Fx モジュールを返します。
func BasicAuthModule() fx.Option {
	return fx.Module("core.basicauth",
		fx.Provide(
			basicauth.NewBasicAuthValidator,
		),
	)
}
