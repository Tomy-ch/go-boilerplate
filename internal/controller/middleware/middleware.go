// Package middleware は、Echoフレームワークのミドルウェアを提供します。
package middleware

import (
	"boilerplate-go/internal/appconfig"
	"boilerplate-go/internal/controller/middleware/echorecover"
	"boilerplate-go/internal/controller/middleware/logging"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// UseMiddlewares は、Echoフレームワークのミドルウェアを設定します。
//
// 副作用としてEchoフレームワークにミドルウェアを登録します。
func UseMiddlewares(
	e *echo.Echo, cfg *appconfig.Config, logger *zap.Logger,
) {
	// ログの設定
	e.Use(logging.Middleware(logger, cfg))
	// パニック復旧の設定
	e.Use(echorecover.Middleware(logger, cfg))
}
