// Package basicauth は、Basic認証に関連するHTTPスタックの機能を提供します。
package basicauth

import (
	"crypto/subtle"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// NewBasicAuthValidator は、Basic認証のバリデータを返します。
func NewBasicAuthValidator(mtcCfg *config.MetricsConfig) echomw.BasicAuthValidator {
	return func(username, password string, _ echo.Context) (bool, error) {
		userOK := subtle.ConstantTimeCompare([]byte(username), []byte(mtcCfg.UserName())) == 1
		passOK := subtle.ConstantTimeCompare([]byte(password), []byte(mtcCfg.Password())) == 1
		return userOK && passOK, nil
	}
}
