package server

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// CustomValidator は Echo に検証器を提供する構造体
type CustomValidator struct {
	validator *validator.Validate
}

// NewValidator は、Echoで使用するための検証器を生成します。
func NewValidator() echo.Validator {
	return &CustomValidator{
		validator: validator.New(),
	}
}

// Validate は echo.Validator インターフェースの実装
func (cv *CustomValidator) Validate(i any) error {
	return cv.validator.Struct(i)
}
