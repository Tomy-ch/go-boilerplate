// Package hook は、outbox relay engine のライフサイクルフックを提供します。
package hook

import (
	"context"

	outboxengine "go-boilerplate/internal/controller/outbox"
	"go-boilerplate/internal/di/lifecycle"
)

// RegisterRelayHooks は、relay engine の poll ループを fx ライフサイクルに結線します。
//   - OnStart: poll ループを detached goroutine で起動する（OnStart はブロックしない）。
//   - OnStop:  engine の context をキャンセルしてループの終了を（stopCtx の範囲で）待つ。
func RegisterRelayHooks(reg lifecycle.Registrar, engine *outboxengine.Engine) {
	// engineCtx は OnStop でのみキャンセルする（OnStart 完了後の startCtx キャンセルに巻き込まれない）。
	engineCtx, cancel := context.WithCancel(context.Background())
	engineDone := make(chan struct{})

	reg.RegisterStart(func(_ context.Context) error {
		go func() {
			defer close(engineDone)
			_ = engine.Run(engineCtx)
		}()
		return nil
	})

	reg.RegisterStop(func(stopCtx context.Context) error {
		cancel()
		select {
		case <-engineDone: // ループ終了
		case <-stopCtx.Done(): // 猶予切れ
		}
		return nil
	})
}
