// Package cors は、CORS（Cross-Origin Resource Sharing）に関するミドルウェアを提供します。
package cors

import (
	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Middleware は、CORSミドルウェアを設定して返します。
func Middleware(secCfg *config.SecurityConfig) echo.MiddlewareFunc {
	return middleware.CORSWithConfig(buildCORSConfig(secCfg.AllowedOrigins()))
}

// buildCORSConfig は、CORSミドルウェアの設定を構築します。
func buildCORSConfig(allowedOrigins []string) middleware.CORSConfig {
	return middleware.CORSConfig{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{
			echo.HEAD,
			echo.GET,
			echo.POST,
			echo.PUT,
			echo.PATCH,
			echo.DELETE,
			echo.OPTIONS,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
		},
		ExposeHeaders: []string{
			echo.HeaderContentDisposition,
			echo.HeaderLocation,
			echo.HeaderXRequestID,
		},
		AllowCredentials: false,
	}
}
