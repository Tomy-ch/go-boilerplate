// Package basicauth は、Basic認証に関連するHTTPスタックの機能を提供します。
package basicauth

import (
	"crypto/subtle"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// NewBasicAuthValidator は、MetricsConfig のユーザ名・パスワードを用いる Basic 認証バリデータを返します。
// タイミング攻撃を防ぐため定数時間比較（subtle.ConstantTimeCompare）を用い、両判定を無条件に評価してから結合して短絡による分岐差も避けます。
// なお資格情報を SHA-256 でハッシュ化してから比較すると長さのタイミング漏れを塞げますが、これは静的な metrics 資格情報に対して軽微であり、パスワードの高速ハッシュ化はセキュリティスキャン（CodeQL）の対象となるため採用しません。
func NewBasicAuthValidator(mtcCfg *config.MetricsConfig) echomw.BasicAuthValidator {
	return func(username, password string, _ echo.Context) (bool, error) {
		userOK := subtle.ConstantTimeCompare([]byte(username), []byte(mtcCfg.UserName())) == 1
		passOK := subtle.ConstantTimeCompare([]byte(password), []byte(mtcCfg.Password())) == 1
		return userOK && passOK, nil
	}
}
