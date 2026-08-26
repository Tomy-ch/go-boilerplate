// Package basicauth は、Basic認証に関連するHTTPスタックの機能を提供します。
package basicauth

import (
	"crypto/subtle"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
)

// NewBasicAuthValidator は、MetricsConfig のユーザ名・パスワードを用いる Basic 認証バリデータを返します。
// タイミング攻撃を防ぐため定数時間比較（subtle.ConstantTimeCompare）を用い、両判定を無条件に評価してから結合して短絡による分岐差も避けます。
func NewBasicAuthValidator(mtcCfg *config.MetricsConfig) echomw.BasicAuthValidator {
	return func(_ *echo.Context, username, password string) (bool, error) {
		userOK := subtle.ConstantTimeCompare([]byte(username), []byte(mtcCfg.UserName())) == 1
		passOK := subtle.ConstantTimeCompare([]byte(password), []byte(mtcCfg.Password())) == 1
		return userOK && passOK, nil
	}
}
