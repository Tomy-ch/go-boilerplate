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
// Echo v5 はサーバーを自身で保持せず [echo.StartConfig] へ集約しますが、StartConfig は
// ブロッキングかつ独自の graceful timeout を持ち、起動と停止を分離する fx のライフサイクルと噛み合いません。
// そのため HTTP サーバーは自前で構築し、タイムアウト設定の置き場も兼ねます。
func NewHTTPServer(e *echo.Echo, srvCfg *config.ServerConfig) *http.Server {
	return &http.Server{
		Handler:           e,
		ReadHeaderTimeout: srvCfg.ReadHeaderTimeout(),
		ReadTimeout:       srvCfg.ReadTimeout(),
		WriteTimeout:      srvCfg.WriteTimeout(),
		IdleTimeout:       srvCfg.IdleTimeout(),
	}
}
