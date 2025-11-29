// Package defaultport は、ポート設定を制御するためのパッケージです。
package defaultport

import (
	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
)

// New は、ポート設定を初期化します。
//
// ポートは本番環境では非表示にします。
func New(e *echo.Echo, appCfg *config.ApplicationConfig) {
	e.HidePort = appCfg.IsAppProductionMode()
}
