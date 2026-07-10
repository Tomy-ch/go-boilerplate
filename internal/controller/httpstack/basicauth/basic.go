// Package basicauth は、Basic認証に関連するHTTPスタックの機能を提供します。
package basicauth

import (
	"crypto/sha256"
	"crypto/subtle"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// NewBasicAuthValidator は、MetricsConfig のユーザ名・パスワードを用いる Basic 認証バリデータを返します。
// タイミング攻撃を防ぐため、両辺を SHA-256 でハッシュ化してから定数時間比較（subtle.ConstantTimeCompare）します。
// 素の資格情報を直接比較すると長さ不一致で即座に不一致判定となり、長さがタイミングから漏れるため、
// 固定長のハッシュ同士を比較してこれを塞ぎます。ユーザ名・パスワードの両判定を無条件に評価してから結合し、短絡による分岐差も避けます。
func NewBasicAuthValidator(mtcCfg *config.MetricsConfig) echomw.BasicAuthValidator {
	return func(username, password string, _ echo.Context) (bool, error) {
		givenUser := sha256.Sum256([]byte(username))
		expectedUser := sha256.Sum256([]byte(mtcCfg.UserName()))
		givenPass := sha256.Sum256([]byte(password))
		expectedPass := sha256.Sum256([]byte(mtcCfg.Password()))

		userOK := subtle.ConstantTimeCompare(givenUser[:], expectedUser[:]) == 1
		passOK := subtle.ConstantTimeCompare(givenPass[:], expectedPass[:]) == 1
		return userOK && passOK, nil
	}
}
