package ctxhelper

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// ErrUnauthenticatedUser は、認証ユーザー情報が取得できない場合のエラーです。
var ErrUnauthenticatedUser = xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")

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

// RequireAuthn は、ctx のスロットから Authn を読みます。未設定の場合は ErrUnauthenticatedUser を返します。
// 認証済み呼び出し元を前提とするハンドラは、GetAuthn を直接呼ばずこちらを使います。
func RequireAuthn(ctx context.Context) (auth.Authn, error) {
	authn, ok := GetAuthn(ctx)
	if !ok {
		return auth.Authn{}, ErrUnauthenticatedUser
	}
	return authn, nil
}

// RequireUserID は、ctx の Authn から内部 UserID を取り出します。
// Authn が未設定の場合は ErrUnauthenticatedUser を、UserID が未解決の場合はその原因をラップして返します。
func RequireUserID(ctx context.Context) (uuid.UUID, error) {
	authn, err := RequireAuthn(ctx)
	if err != nil {
		return uuid.UUID{}, err
	}
	userID, err := authn.UserID()
	if err != nil {
		return uuid.UUID{}, xerrors.Wrap(err, "failed to get user ID from authenticator")
	}
	return userID, nil
}
