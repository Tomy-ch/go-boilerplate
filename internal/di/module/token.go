package module

import (
	"go-boilerplate/internal/infrastructure/token"

	"go.uber.org/fx"
)

// tokenModule は、不透明なトークン文字列の生成を提供するfx.Moduleです。
// clock と同じく、ユースケースがシステム機能（ここでは暗号論的乱数）へ直接依存しないための境界です。
func tokenModule() fx.Option {
	return fx.Module("token",
		fx.Provide(
			token.New,
		),
	)
}
