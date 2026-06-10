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

var ErrLocalMockAuthenticatorInvalidToken = xerrors.Wrap(apperror.ErrUnauthenticated, "local mock authenticator: invalid token")

// authenticator は local 開発用の Authenticator です。
// - verify はしません（ローカル専用）。
// - token の文字列から Subject を決め、Authn を返します。
//
// provider は常に authbd.ProviderMock、Subject 抽出 prefix は常に localPrefix で固定のため、
// 可変設定は持たない。将来 prefix/provider を可変にする要求が出た時点で New(opts...) を導入する。
type authenticator struct{}

// New は local 用 Authenticator のコンストラクタです。
func New() authbd.Authenticator {
	return &authenticator{}
}

// Authenticate は与えられた認証情報を基に認証を行い、認証結果を返します。
// cred は非 nil 前提だが、公開境界の実装として nil でも panic させず無効トークン扱いにする。
func (a *authenticator) Authenticate(_ context.Context, cred *authbd.Credential) (*authbd.Authn, error) {
	if cred == nil {
		return nil, ErrLocalMockAuthenticatorInvalidToken
	}

	// resolveSubject は空文字 or トリム済み非空文字へ正規化するため、ここでの再トリムは不要。
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
