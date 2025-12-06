// Package server は、Echoサーバーの初期化と設定を提供します。
package server

import (
	"boilerplate-go/internal/controller/server"

	"go.uber.org/fx"
)

// ServeModule は、サーバー関連の依存関係を提供するfx.Moduleです。
func ServeModule() fx.Option {
	return fx.Module("server",
		fx.Provide(
			server.NewAppServer,
		),
		fx.Invoke(
			server.ServeHTTP,
		),
	)
}
