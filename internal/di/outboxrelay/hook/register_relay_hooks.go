// Package hook は、outbox relay engine のライフサイクルフックを提供します。
package hook

import (
	"context"

	outboxengine "go-boilerplate/internal/controller/outbox"
	"go-boilerplate/internal/di/lifecycle"
)

// RegisterRelayHooks は、relay engine の poll ループを fx ライフサイクルに結線します。
// 起動・停止契約は [lifecycle.SupervisedRunner] に従います。
func RegisterRelayHooks(reg lifecycle.Registrar, engine *outboxengine.Engine) {
	lifecycle.SupervisedRunner{
		Body: func(ctx context.Context) { _ = engine.Run(ctx) },
	}.Register(reg)
}
