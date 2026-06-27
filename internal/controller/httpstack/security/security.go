// Package security は、セキュリティに関するミドルウェアを提供します。
package security

import (
	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Middleware は、Content-Type-Options / Referrer-Policy / X-Frame-Options / HSTS を設定する Echo セキュリティミドルウェアを返します。各値は SecurityConfig から取得します。
func Middleware(secCfg *config.SecurityConfig) echo.MiddlewareFunc {
	return middleware.SecureWithConfig(buildSecureConfig(secCfg))
}

// buildSecureConfig は、セキュリティ設定を構築します。
func buildSecureConfig(secCfg *config.SecurityConfig) middleware.SecureConfig {
	return middleware.SecureConfig{
		ContentTypeNosniff:    secCfg.ContentTypeNosniff(),
		ReferrerPolicy:        secCfg.ReferrerPolicy(),
		XFrameOptions:         secCfg.XFrameOptions(),
		HSTSMaxAge:            int(secCfg.HSTSMaxAge().Seconds()),
		HSTSExcludeSubdomains: secCfg.HSTSExcludeSubdomains(),
		HSTSPreloadEnabled:    secCfg.HSTSPreloadEnabled(),
	}
}
