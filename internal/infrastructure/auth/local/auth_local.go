// Package local はローカル開発用の認証基盤実装を提供します。
package local

import (
	"context"
	"strings"

	"go-boilerplate/internal/apperror"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/uuid"
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

	sub := a.resolveSubject(cred.Token())
	if sub == "" {
		return nil, ErrLocalMockAuthenticatorInvalidToken
	}

	authn, err := authbd.New(sub, authbd.IssuerMock, nil, nil)
	if err != nil {
		return nil, err
	}

	// IdentityResolver が配線されるまでのローカル専用の暫定措置。
	// subject を UUID として解釈できる場合に限り内部 UserID として解決する。
	if id, perr := uuid.Parse(sub); perr == nil {
		authn = authn.WithUserID(id)
	}

	return authn, nil
}

// resolveSubject は token から localPrefix を除いた Subject を抽出します。
func (a *authenticator) resolveSubject(token string) string {
	if after, ok := strings.CutPrefix(token, localPrefix); ok {
		sub := strings.TrimSpace(after)
		if sub != "" {
			return sub
		}
	}

	return ""
}
