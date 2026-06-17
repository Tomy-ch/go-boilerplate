// Package local はローカル開発用の認証基盤実装を提供します。
package local

import (
	"context"
	"strings"

	"go-boilerplate/internal/apperror"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/xerrors"
)

const (
	localPrefix = "debug:"
)

// ErrLocalMockAuthenticatorInvalidToken は、ローカルモック認証でトークンが不正な場合のエラーです。
var ErrLocalMockAuthenticatorInvalidToken = xerrors.Wrap(apperror.ErrUnauthenticated, "local mock authenticator: invalid token")

// authenticator は local 開発用の Authenticator です。
// - verify はしません（ローカル専用）。
// - token の文字列から Subject を決め、Authn を返します。
type authenticator struct{}

// New は local 用 Authenticator のコンストラクタです。
func New() authbd.Authenticator {
	return &authenticator{}
}

// Authenticate は与えられた認証情報を基に認証を行い、認証結果を返します。
// cred が nil の場合は無効トークンとして扱います。
func (a *authenticator) Authenticate(_ context.Context, cred *authbd.Credential) (*authbd.Authn, error) {
	if cred == nil {
		return nil, ErrLocalMockAuthenticatorInvalidToken
	}

	sub := a.resolveSubject(cred.AccessToken())
	if sub == "" {
		return nil, ErrLocalMockAuthenticatorInvalidToken
	}

	return authbd.New(
		sub,
		authbd.ProviderMock,
		nil,
		nil,
	)
}

// resolveSubject は token から localPrefix を除いた Subject を抽出します。
func (a *authenticator) resolveSubject(token string) string {
	if strings.HasPrefix(token, localPrefix) {
		sub := strings.TrimSpace(strings.TrimPrefix(token, localPrefix))
		if sub != "" {
			return sub
		}
	}

	return ""
}
