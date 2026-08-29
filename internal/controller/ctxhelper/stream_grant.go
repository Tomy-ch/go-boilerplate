package ctxhelper

import (
	"context"

	"go-boilerplate/internal/apperror"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// ErrStreamGrantMissing は、検証済みの stream ticket が無い場合のエラーです。
var ErrStreamGrantMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "requires a verified stream ticket")

type streamGrantSlotKey struct{}

// streamGrantSlot は、認証前に仕込んで後段ハンドラと共有する、検証済み ticket の束縛（StreamGrant）の可変スロット。
// 成功のみを保持し、失敗は Authn スロットが運ぶ（README「Provided helpers」）。
type streamGrantSlot struct {
	grant rt.StreamGrant
	set   bool
	// revalidate は、同じ ticket をもう一度検証し直す口。資格情報の生値は認証器の中に閉じたままで、
	// 後段へ渡るのはこの関数値だけ。
	revalidate func(ctx context.Context) error
}

// WithStreamGrant は、空の StreamGrant スロットを ctx に仕込む。
func WithStreamGrant(ctx context.Context) context.Context {
	return context.WithValue(ctx, streamGrantSlotKey{}, &streamGrantSlot{})
}

// SetStreamGrant は、ctx のスロットへ検証済み ticket を書き込む。スロットが無ければ false。
func SetStreamGrant(ctx context.Context, grant rt.StreamGrant) bool {
	slot, ok := ctx.Value(streamGrantSlotKey{}).(*streamGrantSlot)
	if !ok {
		return false
	}
	slot.grant, slot.set = grant, true
	return true
}

// GetStreamGrant は、ctx のスロットから検証済み ticket を読む。未設定なら ok=false。
func GetStreamGrant(ctx context.Context) (rt.StreamGrant, bool) {
	slot, ok := ctx.Value(streamGrantSlotKey{}).(*streamGrantSlot)
	if !ok || !slot.set {
		return rt.StreamGrant{}, false
	}
	return slot.grant, true
}

// RequireStreamGrant は、ctx のスロットから検証済み ticket を読みます。未設定の場合は ErrStreamGrantMissing を返します。
// stream ticket による認証を前提とするハンドラは、GetStreamGrant を直接呼ばずこちらを使います。
func RequireStreamGrant(ctx context.Context) (rt.StreamGrant, error) {
	grant, ok := GetStreamGrant(ctx)
	if !ok {
		return rt.StreamGrant{}, ErrStreamGrantMissing
	}
	return grant, nil
}

// SetStreamRevalidator は、ctx のスロットへ ticket の再検証手段を書き込む。スロットが無ければ false。
//
// 接続の確立は「ticket 検証 →（外部 I/O）→ 接続の索引へ登録」の 2 段階で、その間に届いた失効通知は
// まだ索引に無い接続を取りこぼす。登録の直後にもう一度検証できれば、検証が無効化より後なら拒否され、
// 前なら登録も無効化より前なので後続の失効通知が拾える（ADR-0074 の「in-service の失効は即時」）。
func SetStreamRevalidator(ctx context.Context, revalidate func(ctx context.Context) error) bool {
	slot, ok := ctx.Value(streamGrantSlotKey{}).(*streamGrantSlot)
	if !ok {
		return false
	}
	slot.revalidate = revalidate
	return true
}

// GetStreamRevalidator は、ctx のスロットから ticket の再検証手段を読む。未設定なら ok=false。
func GetStreamRevalidator(ctx context.Context) (func(ctx context.Context) error, bool) {
	slot, ok := ctx.Value(streamGrantSlotKey{}).(*streamGrantSlot)
	if !ok || slot.revalidate == nil {
		return nil, false
	}
	return slot.revalidate, true
}
