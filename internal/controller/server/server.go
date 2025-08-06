// Package server は、Echoサーバーの初期化と設定を提供します。
package server

import (
	"boilerplate-go/internal/appconfig"
	"boilerplate-go/internal/controller/middleware/logging"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func New(
	cfg *appconfig.Config,
	logger *zap.Logger,
) *echo.Echo {
	e := echo.New()
	e.Use(logging.Middleware(logger, cfg))

	return e
}
