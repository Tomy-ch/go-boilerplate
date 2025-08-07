package server

import "github.com/labstack/echo/v4"

// NewBinder は、EchoでBind()使用するための実体を生成します。
func NewBinder() echo.Binder {
	return &echo.DefaultBinder{}
}
