package module

import (
	"go-boilerplate/internal/infrastructure/security"

	"go.uber.org/fx"
)

// securityModule は、パスワードハッシュ化などのセキュリティ関連の依存を提供するfx.Moduleです。
func securityModule() fx.Option {
	return fx.Module("security",
		fx.Provide(
			security.NewBcryptHasher,
		),
	)
}
