//go:generate mockgen -source=$GOFILE -destination=mock/mock_authenticator.gen.go -package=mock_$GOPACKAGE
package auth

import (
	"context"
)

// Authenticator は認証を行うためのインターフェースです。
type Authenticator interface {
	// Authenticate は与えられた認証情報を基に認証を行い、認証結果を返します。
	Authenticate(ctx context.Context, cred *Credential) (*Authn, error)
}
