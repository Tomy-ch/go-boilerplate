// Package core は、HTTP ミドルウェア層で使用する Fx モジュール群を提供します。
// 認証（AuthnModule）・リクエストバリデーション（ValidatorModule）・セキュリティ Cookie
// （SecurityCookieModule）・スキッパー（SkipperModule）・Basic 認証（BasicAuthModule）を含みます。
package core

import (
	"go-boilerplate/internal/controller/httpstack/oapi/validator"

	"go.uber.org/fx"
)

// ValidatorModule は、ルーティング時に自動で解決されるバリデーションのコア機能部分を提供するfxモジュールを返します。
func ValidatorModule() fx.Option {
	return fx.Module("core.validator",
		fx.Provide(
			validator.GetValidator,
		),
	)
}
