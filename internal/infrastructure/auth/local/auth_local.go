// Package local はローカル開発用の認証基盤実装を提供します。
package local

import (
	"context"
	"strings"

	"boilerplate-go/internal/apperror"
	authbd "boilerplate-go/internal/usecase/boundary/auth"
	"boilerplate-go/pkg/xerrors"
)

const (
	localPrefix = "debug:"
)

var ErrLocalMockAuthenticatorInvalidToken = xerrors.Wrap(apperror.ErrUnauthenticated, "local mock authenticator: invalid token")

// config は local 用の Authenticator 設定です。
// verify（署名検証等）は行わず、入力トークンから Authn を組み立てます。
type config struct {
	// provider は Authn.provider に入れる値（例: "mock"）
	provider string

	// prefix は token から Subject を抽出するための prefix。
	prefix string
}

// authenticator は local 開発用の Authenticator です。
// - verify はしません（ローカル専用）。
// - token の文字列から Subject を決め、Authn を返します。
type authenticator struct {
	cfg *config
}

// New は LocalMockAuthenticator のコンストラクタです。
func New() authbd.Authenticator {
	return &authenticator{cfg: &config{
		provider: authbd.ProviderMock,
		prefix:   localPrefix,
	}}
}

// Authenticate は与えられた認証情報を基に認証を行い、認証結果を返します。
func (a *authenticator) Authenticate(_ context.Context, cred *authbd.Credential) (*authbd.Authn, error) {
	sub := a.resolveSubject(cred.AccessToken())
	if strings.TrimSpace(sub) == "" {
		return nil, ErrLocalMockAuthenticatorInvalidToken
	}

	return authbd.New(
		sub,
		a.cfg.provider,
		nil,
		nil,
	)
}

// resolveSubject は token から Subject を抽出します。
func (a *authenticator) resolveSubject(token string) string {
	if a.cfg.prefix != "" && strings.HasPrefix(token, a.cfg.prefix) {
		sub := strings.TrimSpace(strings.TrimPrefix(token, a.cfg.prefix))
		if sub != "" {
			return sub
		}
	}

	return ""
}
