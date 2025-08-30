// Package ipextractor は、IPアドレスの抽出に関する機能を提供します。
package ipextractor

import (
	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
)

func New(e *echo.Echo, cfg *config.Config) {
	e.IPExtractor = NewIPExtractor(cfg)
}

// NewIPExtractor は、EchoでクライアントのIPアドレスを抽出するためのインスタンスを生成します。
//
//	本番環境ではX-Forwarded-ForヘッダーからIPアドレスを抽出し、開発環境では直接抽出します。
//	その他の環境では、本番に準じてX-Forwarded-Forヘッダーから抽出します。
func NewIPExtractor(cfg *config.Config) echo.IPExtractor {
	switch {
	case cfg.IsAppProductionMode():
		return echo.ExtractIPFromXFFHeader(
			echo.TrustIPRange(cfg.CIDR()),
		)
	case cfg.IsAppDevelopmentMode():
		return echo.ExtractIPDirect()
	default:
		// 安全のため、本番に準じてIPアドレスを抽出する
		return echo.ExtractIPFromXFFHeader(
			echo.TrustIPRange(cfg.CIDR()),
		)
	}
}
