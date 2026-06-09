// Package binder は、Echo の Bind() で使う echo.Binder を提供します。
package binder

import (
	"github.com/labstack/echo/v4"
)

// New は、生成した Binder を Echo に設定します。
func New(e *echo.Echo) {
	e.Binder = NewBinder()
}

// NewBinder は、Echo の Bind() で使う echo.Binder を生成します。
func NewBinder() echo.Binder {
	return &echo.DefaultBinder{}
}
