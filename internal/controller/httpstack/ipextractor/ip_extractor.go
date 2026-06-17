// Package ipextractor は、IPアドレスの抽出に関する機能を提供します。
package ipextractor

import (
	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
)

// New は、アプリケーション設定とセキュリティ設定をもとにIPアドレス抽出器を生成し、Echoインスタンスに設定します。
func New(e *echo.Echo, appCfg *config.ApplicationConfig, secCfg *config.SecurityConfig) {
	e.IPExtractor = NewIPExtractor(appCfg, secCfg)
}

// NewIPExtractor は、EchoでクライアントのIPアドレスを抽出するためのインスタンスを生成します。
//
//	開発環境では直接抽出し、それ以外（本番・未知環境）ではX-Forwarded-Forヘッダーから抽出します。
func NewIPExtractor(appCfg *config.ApplicationConfig, secCfg *config.SecurityConfig) echo.IPExtractor {
	if appCfg.IsDevelopmentMode() {
		return echo.ExtractIPDirect()
	}
	return echo.ExtractIPFromXFFHeader(
		echo.TrustIPRange(secCfg.CIDR()),
	)
}
