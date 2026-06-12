package ctxhelper

import (
	"context"

	"go-boilerplate/internal/usecase/boundary/auth"
)

type authnSlotKey struct{}

// authnSlot は、認証前に仕込んで後段ハンドラと共有する Authn の可変スロット。
type authnSlot struct {
	authn auth.Authn
	set   bool
}

// WithAuthn は、空の Authn スロットを ctx に仕込む。
func WithAuthn(ctx context.Context) context.Context {
	return context.WithValue(ctx, authnSlotKey{}, &authnSlot{})
}

// SetAuthn は、ctx のスロットへ Authn を書き込む。スロットが無ければ false。
func SetAuthn(ctx context.Context, authn auth.Authn) bool {
	slot, ok := ctx.Value(authnSlotKey{}).(*authnSlot)
	if !ok {
		return false
	}
	slot.authn, slot.set = authn, true
	return true
}

// GetAuthn は、ctx のスロットから Authn を読む。未設定なら ok=false。
func GetAuthn(ctx context.Context) (auth.Authn, bool) {
	slot, ok := ctx.Value(authnSlotKey{}).(*authnSlot)
	if !ok || !slot.set {
		return auth.Authn{}, false
	}
	return slot.authn, true
}
