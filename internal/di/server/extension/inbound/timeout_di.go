package inbound

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/timeout"
	"go-boilerplate/internal/di/server/extension"

	"go.uber.org/fx"
)

// timeoutPrePriority は、timeout ミドルウェアの Pre 適用順序です。
// priority が小さいほど先に実行されるため、uri(=1) の直後(=2)に位置し、
// 全 Use / openapi / handler / DB を単一の deadline budget で覆います。
const timeoutPrePriority = 2

// TimeoutModule は、リクエスト deadline budget のミドルウェアを提供する fx モジュールを返します。
func TimeoutModule() fx.Option {
	return fx.Module("mw.timeout",
		fx.Provide(
			TimeoutPreMiddleware,
		),
	)
}

// TimeoutPreMiddleware は、SERVER_REQUEST_TIMEOUT を deadline budget とする
// timeout ミドルウェアを Pre ミドルウェアとして提供します。
func TimeoutPreMiddleware(srvCfg *config.ServerConfig) extension.PreMiddlewareOut {
	return extension.PreMiddlewareOut{
		Middleware: extension.PreMiddleware{
			Name:       "timeout",
			Priority:   timeoutPrePriority,
			Middleware: timeout.Middleware(srvCfg.RequestTimeout()),
		},
	}
}
