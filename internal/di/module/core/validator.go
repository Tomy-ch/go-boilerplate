// Package core は、モジュールのコア機能部分を提供します。
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
