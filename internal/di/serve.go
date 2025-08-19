// Package di は、アプリケーションの設定をDI用に提供します。
package di

import (
	"boilerplate-go/internal/controller/handler"
	"boilerplate-go/internal/controller/server"

	"go.uber.org/fx"
)

// ServeModule は、サーバー関連の依存関係を提供するfx.Moduleです。
func ServeModule() fx.Option {
	return fx.Module("server",
		fx.Provide(
			server.New,
			handler.MountRoutes,
		),
		fx.Invoke(
			server.ServeHTTP,
		),
	)
}
