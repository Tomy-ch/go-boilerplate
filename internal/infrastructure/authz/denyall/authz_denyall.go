// Package denyall は、すべての認可要求を拒否する fail-closed な Authorizer 実装を提供します。
// サンプルの認可実装（userrole）を剥がした後の安全な既定として据えられ、
// 認可にオプトインした usecase が独自の Authorizer を挿すまで、すべての要求を拒否します。
package denyall

import (
	"context"

	authbd "go-boilerplate/internal/usecase/boundary/auth"
	authzbd "go-boilerplate/internal/usecase/boundary/authz"
)

// authorizer は、すべての認可要求を拒否する fail-closed な Authorizer です。
type authorizer struct{}

// New は、全拒否 Authorizer のコンストラクタです。
func New() authzbd.Authorizer {
	return &authorizer{}
}

// Authorize は、authn / action / resource によらず常に拒否（ErrForbidden）を返します。
func (a *authorizer) Authorize(_ context.Context, _ *authbd.Authn, _ authzbd.Action, _ *authzbd.Resource) error {
	return authzbd.ErrForbidden
}
