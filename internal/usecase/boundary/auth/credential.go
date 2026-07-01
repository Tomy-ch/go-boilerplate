package auth

import "strings"

// Credential は認証情報（トークン）を表します。
type Credential struct {
	accessToken string // アクセストークン
}

// NewCredential は、アクセストークンを検証して Credential を生成して返します。
// トークンが空文字列またはホワイトスペースのみの場合は ErrTokenMissing を返します。
func NewCredential(
	accessToken string,
) (*Credential, error) {
	trimmedToken := strings.TrimSpace(accessToken)
	if trimmedToken == "" {
		return nil, ErrTokenMissing
	}

	return &Credential{
		accessToken: trimmedToken,
	}, nil
}

// AccessToken はアクセストークンを返します。
func (c *Credential) AccessToken() string {
	return c.accessToken
}
