// Package validator は、リクエストの検証を行うミドルウェアを提供します。
package validator

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// Validator は Echo に検証器を提供する構造体
type Validator struct {
	validator *validator.Validate
}

// New は、Echo に検証器を提供するための関数です。
func New(e *echo.Echo) {
	e.Validator = NewValidator()
}

// NewValidator は、Echoで使用するための検証器を生成します。
func NewValidator() echo.Validator {
	return &Validator{
		validator: validator.New(),
	}
}

// Validate は echo.Validator インターフェースの実装
func (cv *Validator) Validate(i any) error {
	return cv.validator.Struct(i)
}
