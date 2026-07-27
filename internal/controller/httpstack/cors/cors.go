// Package cors は、CORS（Cross-Origin Resource Sharing）に関するミドルウェアを提供します。
package cors

import (
	"net/http"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// corsMaxAgeSeconds は、プリフライト結果のキャッシュ秒数。
const corsMaxAgeSeconds = 600

// Middleware は、CORSミドルウェアを設定して返します。
func Middleware(secCfg *config.SecurityConfig) echo.MiddlewareFunc {
	return middleware.CORSWithConfig(buildCORSConfig(secCfg.AllowedOrigins()))
}

// buildCORSConfig は、CORSミドルウェアの設定を構築します。
func buildCORSConfig(allowedOrigins []string) middleware.CORSConfig {
	return middleware.CORSConfig{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
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
		MaxAge:           corsMaxAgeSeconds,
	}
}
