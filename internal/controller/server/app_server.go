// Package server は、サーバーインスタンスを初期化します。
package server

import (
	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
)

// NewAppServer は、サーバーインスタンスを作成します。
func NewAppServer(srvCfg *config.ServerConfig) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Server.ReadHeaderTimeout = srvCfg.ReadHeaderTimeout()
	e.Server.ReadTimeout = srvCfg.ReadTimeout()
	e.Server.WriteTimeout = srvCfg.WriteTimeout()
	e.Server.IdleTimeout = srvCfg.IdleTimeout()
	return e
}
