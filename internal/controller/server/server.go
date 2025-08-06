// Package server は、Echoサーバーの初期化と設定を提供します。
package server

import (
	"github.com/labstack/echo/v4"
)

func New() *echo.Echo {
	return echo.New()
}
