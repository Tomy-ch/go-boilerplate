// Package security は、セキュリティに関するミドルウェアを提供します。
package security

import (
	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Middleware は、セキュリティミドルウェアを構築します。
func Middleware(appCfg *config.ApplicationConfig) echo.MiddlewareFunc {
	return middleware.SecureWithConfig(buildSecureConfig(appCfg))
}

// buildSecureConfig は、セキュリティ設定を構築します。
func buildSecureConfig(appCfg *config.ApplicationConfig) middleware.SecureConfig {
	scfg := middleware.SecureConfig{
		XSSProtection:      "",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "DENY",
		ReferrerPolicy:     "no-referrer",
	}

	if appCfg.IsProductionMode() {
		scfg.HSTSExcludeSubdomains = false
		scfg.HSTSMaxAge = 31536000
	}

	return scfg
}
