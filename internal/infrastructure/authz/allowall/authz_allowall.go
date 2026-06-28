// Package allowall は、すべての操作を許可する開発用の Authorizer 実装を提供します。
package allowall

import (
	"context"

	authbd "go-boilerplate/internal/usecase/boundary/auth"
	authzbd "go-boilerplate/internal/usecase/boundary/authz"
)

// authorizer は、すべての認可要求を許可する Authorizer です（ローカル/開発用の割り切り実装）。
// 本番でポリシー制御が必要になった際に、RBAC / 外部ポリシーエンジン実装へ差し替えます。
type authorizer struct{}

// New は、全許可 Authorizer のコンストラクタです。
func New() authzbd.Authorizer {
	return &authorizer{}
}

// Authorize は、action / resource によらず常に許可（nil）を返します。
func (a *authorizer) Authorize(_ context.Context, _ *authbd.Authn, _ authzbd.Action, _ *authzbd.Resource) error {
	return nil
}
