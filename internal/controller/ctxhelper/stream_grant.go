package ctxhelper

import (
	"context"

	"go-boilerplate/internal/apperror"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	"go-boilerplate/pkg/xerrors"
)

// ErrStreamGrantMissing は、検証済みの stream ticket が無い場合のエラーです。
var ErrStreamGrantMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "requires a verified stream ticket")

type streamGrantSlotKey struct{}

// streamGrantSlot は、認証前に仕込んで後段ハンドラと共有する、検証済み stream ticket の可変スロット。
// 失敗は Authn スロットが運ぶ（stream ticket の不備も認証の失敗であり、拒否へ結びつける経路は同じ）。
type streamGrantSlot struct {
	grant ucrealtime.VerifiedTicketView
	set   bool
}

// WithStreamGrant は、空の StreamGrant スロットを ctx に仕込む。
func WithStreamGrant(ctx context.Context) context.Context {
	return context.WithValue(ctx, streamGrantSlotKey{}, &streamGrantSlot{})
}

// SetStreamGrant は、ctx のスロットへ検証済み ticket を書き込む。スロットが無ければ false。
func SetStreamGrant(ctx context.Context, grant ucrealtime.VerifiedTicketView) bool {
	slot, ok := ctx.Value(streamGrantSlotKey{}).(*streamGrantSlot)
	if !ok {
		return false
	}
	slot.grant, slot.set = grant, true
	return true
}

// GetStreamGrant は、ctx のスロットから検証済み ticket を読む。未設定なら ok=false。
func GetStreamGrant(ctx context.Context) (ucrealtime.VerifiedTicketView, bool) {
	slot, ok := ctx.Value(streamGrantSlotKey{}).(*streamGrantSlot)
	if !ok || !slot.set {
		return ucrealtime.VerifiedTicketView{}, false
	}
	return slot.grant, true
}

// RequireStreamGrant は、ctx のスロットから検証済み ticket を読みます。未設定の場合は ErrStreamGrantMissing を返します。
// stream ticket による認証を前提とするハンドラは、GetStreamGrant を直接呼ばずこちらを使います。
func RequireStreamGrant(ctx context.Context) (ucrealtime.VerifiedTicketView, error) {
	grant, ok := GetStreamGrant(ctx)
	if !ok {
		return ucrealtime.VerifiedTicketView{}, ErrStreamGrantMissing
	}
	return grant, nil
}
