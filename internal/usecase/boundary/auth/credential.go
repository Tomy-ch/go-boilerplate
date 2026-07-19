package auth

import "strings"

const (
	// SchemeBearer は Bearer 認証スキームを示します。
	SchemeBearer = "Bearer"
)

// Credential は認証情報を表します。
// スキーム（Bearer 等）とトークンを持つ、認証方式に依存しない中立表現です。
type Credential struct {
	scheme string // 認証スキーム（例: "Bearer"）
	token  string // トークン
}

// NewCredential は、スキームとトークンから Credential を生成して返します。
// トークンが空文字列またはホワイトスペースのみの場合は ErrTokenMissing を返します。
func NewCredential(
	scheme string,
	token string,
) (*Credential, error) {
	trimmedToken := strings.TrimSpace(token)
	if trimmedToken == "" {
		return nil, ErrTokenMissing
	}

	return &Credential{
		scheme: strings.TrimSpace(scheme),
		token:  trimmedToken,
	}, nil
}

// Scheme は認証スキームを返します。
func (c *Credential) Scheme() string {
	return c.scheme
}

// Token はトークンを返します。
func (c *Credential) Token() string {
	return c.token
}
