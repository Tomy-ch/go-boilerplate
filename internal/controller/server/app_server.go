// Package server は、Echo サーバーの初期化と、HTTP リクエストのログ入力・クエリ/パスパラメータ抽出ユーティリティを提供します。
package server

import (
	"net/http"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v5"
)

// NewAppServer は、サーバーインスタンスを作成します。
func NewAppServer() *echo.Echo {
	return echo.New()
}

// NewHTTPServer は、Echo をハンドラとする HTTP サーバーを作成します。
// Echo v5 の [echo.StartConfig] を使わない理由は ADR-0020 (echo-http-framework) を参照。
func NewHTTPServer(e *echo.Echo, srvCfg *config.ServerConfig) *http.Server {
	return &http.Server{
		Handler:           e,
		ReadHeaderTimeout: srvCfg.ReadHeaderTimeout(),
		ReadTimeout:       srvCfg.ReadTimeout(),
		WriteTimeout:      srvCfg.WriteTimeout(),
		IdleTimeout:       srvCfg.IdleTimeout(),
	}
}
