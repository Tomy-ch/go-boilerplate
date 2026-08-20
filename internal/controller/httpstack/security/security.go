// Package security は、セキュリティに関するミドルウェアを提供します。
package security

import (
	"net/http"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// headerCrossOriginResourcePolicy は、Cross-Origin-Resource-Policy ヘッダーの名前です。
// echo の Secure ミドルウェアはこのヘッダーを扱わないため、名前を自前で持ちます。
const headerCrossOriginResourcePolicy = "Cross-Origin-Resource-Policy"

// Middleware は、Content-Type-Options / Referrer-Policy / X-Frame-Options / HSTS /
// Cross-Origin-Resource-Policy を設定する Echo セキュリティミドルウェアを返します。
// 各値は SecurityConfig から取得します。
func Middleware(secCfg *config.SecurityConfig) echo.MiddlewareFunc {
	secure := middleware.SecureWithConfig(buildSecureConfig(secCfg))
	corp := secCfg.CrossOriginResourcePolicy()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return secure(func(c *echo.Context) error {
			applyCORP(c.Response().Header(), corp)
			return next(c)
		})
	}
}

// applyCORP は、Cross-Origin-Resource-Policy ヘッダーを設定します。
// policy が空ならヘッダーを設定しません。空文字は「このヘッダーを出さない」という設定値であり、
// 値の無いヘッダーを送ることとは区別されます。
func applyCORP(h http.Header, policy string) {
	if policy == "" {
		return
	}
	h.Set(headerCrossOriginResourcePolicy, policy)
}

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
