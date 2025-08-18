// Package binder は、Bind()の仕様を制御するミドルウェアを提供します。
package binder

import (
	"github.com/labstack/echo/v4"
)

// New は、EchoでBind()使用するための実体を生成します。
func New(e *echo.Echo) {
	e.Binder = NewBinder()
}

// NewBinder は、EchoでBind()使用するための実体を生成します。
func NewBinder() echo.Binder {
	return &echo.DefaultBinder{}
}
