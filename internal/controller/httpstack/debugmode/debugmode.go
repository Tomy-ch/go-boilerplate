// Package debugmode は、アプリケーションのデバッグモードを制御するためのパッケージです。
package debugmode

import (
	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
)

// New は、開発モード向けのデバッグ支援機能を設定します。
func New(e *echo.Echo, appCfg *config.ApplicationConfig) {
	e.Debug = appCfg.IsDevelopmentMode()
}
