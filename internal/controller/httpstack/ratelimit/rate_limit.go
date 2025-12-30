// Package ratelimit は、IPアドレスごとのレートリミッターを提供します。
package ratelimit

import (
	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
)

func Middleware(rl IPRateLimiter, ipCfg *config.IPRateLimitConfig) echo.MiddlewareFunc {
	if !ipCfg.Enabled() {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !rl.AllowRequest(c) {
				c.Response().Header().Set("Retry-After", "1")
				return apperror.ErrTooManyRequests
			}
			return next(c)
		}
	}
}
