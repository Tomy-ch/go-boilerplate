//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package authz は認可（authorization）を行うためのインターフェースを提供します。
package authz

import (
	"context"

	"go-boilerplate/internal/usecase/boundary/auth"
)

// Authorizer は認可判断を行うためのインターフェースです。
type Authorizer interface {
	// Authorize は、認証主体 authn が resource に対して action を実行してよいかを判定します。
	// 許可する場合は nil を、拒否する場合は apperror.ErrPermissionDenied でラップしたエラー（ErrForbidden）を返します。
	Authorize(ctx context.Context, authn *auth.Authn, action Action, resource *Resource) error
}
