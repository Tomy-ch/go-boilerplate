package server

import (
	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
)

// NewIPExtractor は、EchoでクライアントのIPアドレスを抽出するためのインスタンスを生成します。
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
