// Package basicauth は、Basic認証に関連するHTTPスタックの機能を提供します。
package basicauth

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// NewBasicAuthValidator は、Basic認証のバリデータを返します。
func NewBasicAuthValidator(mtcCfg *config.MetricsConfig) echomw.BasicAuthValidator {
	return func(username, password string, _ echo.Context) (bool, error) {
		if username == mtcCfg.UserName() && password == mtcCfg.Password() {
			return true, nil
		}
		return false, apperror.ErrUnauthenticated
	}
}
