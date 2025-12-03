// Package banner は、バナー表示を制御するためのパッケージです。
package banner

import (
	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
)

// New は、バナーの表示設定を行います。
//
// バナーは本番環境では非表示にする
func New(e *echo.Echo, appCfg *config.ApplicationConfig) {
	e.HideBanner = appCfg.IsProductionMode()
}
