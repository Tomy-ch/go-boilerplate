package security

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack/ratelimit"
	"boilerplate-go/internal/di/server/extension"

	"go.uber.org/fx"
)

const rateLimitPriority = 9

// RateLimitModule は、レートリミット制御のミドルウェアを提供するfxモジュールを返します。
func RateLimitModule() fx.Option {
	return fx.Module("mw.ratelimit",
		fx.Provide(
			RateLimitMiddleware,
		),
	)
}

// RateLimitMiddleware は、レートリミット制御のミドルウェアを生成します。
func RateLimitMiddleware(rl ratelimit.IPRateLimiter, ipCfg *config.IPRateLimitConfig) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "ratelimit",
			Priority:   rateLimitPriority,
			Middleware: ratelimit.Middleware(rl, ipCfg),
		},
	}
}
