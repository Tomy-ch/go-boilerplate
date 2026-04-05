package core

import (
	"go-boilerplate/internal/controller/httpstack/ratelimit"

	"go.uber.org/fx"
)

// IPRateLimiterModule は、IPレートリミッターのコア機能部分を提供するfxモジュールを返します。
func IPRateLimiterModule() fx.Option {
	return fx.Module("core.ip_rate_limiter",
		fx.Provide(
			ratelimit.NewIPRateLimiter,
		),
	)
}
