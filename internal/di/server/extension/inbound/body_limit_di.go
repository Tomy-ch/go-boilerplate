package inbound

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/bodylimit"
	"go-boilerplate/internal/di/server/extension"

	"go.uber.org/fx"
)

// bodyLimitPrePriority は、body limit ミドルウェアの Pre 適用順序です。
// OpenAPI validator（Use=6）が requestBody を読む前に確実に上限を適用するため Pre に置きます。
// uri=1 の次。deadline budget の timeout（Pre=3）より外側で、body 読み取りも budget 内に収まります。
const bodyLimitPrePriority = 2

// BodyLimitModule は、リクエストボディ上限のミドルウェアを提供する fx モジュールを返します。
func BodyLimitModule() fx.Option {
	return fx.Module("mw.bodylimit",
		fx.Provide(
			BodyLimitPreMiddleware,
		),
	)
}

// BodyLimitPreMiddleware は、SERVER_BODY_LIMIT_MB を上限とする body limit ミドルウェアを
// Pre ミドルウェアとして提供します。
func BodyLimitPreMiddleware(srvCfg *config.ServerConfig) extension.PreMiddlewareOut {
	return extension.PreMiddlewareOut{
		Middleware: extension.PreMiddleware{
			Name:       "bodyLimit",
			Priority:   bodyLimitPrePriority,
			Middleware: bodylimit.Middleware(srvCfg.BodyLimitMB()),
		},
	}
}
