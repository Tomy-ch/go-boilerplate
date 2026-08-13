//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package auth

import (
	"context"
)

// IdentityResolver は、認証済みの外部アイデンティティ（issuer + subject）を内部ユーザーへ解決するインターフェースです。
// 認証成功後に適用します。
type IdentityResolver interface {
	// Resolve は、authn の Issuer と Subject から内部ユーザーを解決し、UserID を設定した Authn を返します。
	// 対応する内部ユーザーが存在しない、または利用できない状態（削除済み等）の場合は認証エラーを返します。
	Resolve(ctx context.Context, authn *Authn) (*Authn, error)
}
