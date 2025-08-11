// Package middleware は、Echoフレームワークのミドルウェアを提供します。
package middleware

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/middleware/cors"
	"boilerplate-go/internal/controller/middleware/logging"
	"boilerplate-go/internal/controller/middleware/recovery"
	"boilerplate-go/internal/controller/middleware/requestid"
	"boilerplate-go/internal/controller/middleware/uricontrol"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// UseMiddlewares は、Echoフレームワークのミドルウェアを設定します。
//
// 副作用としてEchoフレームワークにミドルウェアを登録します。
func UseMiddlewares(
	e *echo.Echo, cfg *config.Config, logger *zap.Logger,
) {
	// URI制御の設定
	e.Pre(uricontrol.Middleware())
	// ログの設定
	e.Use(logging.Middleware(logger))
	// パニック復旧の設定
	e.Use(recovery.Middleware(logger, cfg))
	// リクエストIDの設定
	e.Use(requestid.Middleware())
	// CORSの設定
	e.Use(cors.Middleware(cfg))
}
