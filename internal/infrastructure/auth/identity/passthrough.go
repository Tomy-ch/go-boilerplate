// Package identity は、外部アイデンティティ解決（IdentityResolver）の substrate 既定実装を提供します。
package identity

import (
	"context"

	authbd "go-boilerplate/internal/usecase/boundary/auth"
)

// passthroughResolver は、内部ユーザー解決を行わない既定の IdentityResolver です。
// substrate はユーザーストアを持たないため、認証済みの Authn を UserID 未解決のまま通します。
// プロジェクト固有のユーザーストアへの解決は、本実装を差し替えて提供してください。
type passthroughResolver struct{}

// New は、UserID を解決しない passthrough な IdentityResolver を返します。
// ユーザーストアを持たない substrate 向けの既定実装です。
func New() authbd.IdentityResolver {
	return &passthroughResolver{}
}

// Resolve は、認証済みの Authn を UserID 未解決のまま返します（内部ユーザーは解決しません）。
func (passthroughResolver) Resolve(_ context.Context, authn *authbd.Authn) (*authbd.Authn, error) {
	return authn, nil
}
