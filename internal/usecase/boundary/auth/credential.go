package auth

// Credential は認証情報（トークン）を表します。
// 今回は access token のみ。将来 mTLS/DPoP 等が増えてもここを拡張すればOK。
type Credential struct {
	accessToken string // アクセストークン
}

func NewCredential(
	accessToken string,
) (*Credential, error) {
	if accessToken == "" {
		return nil, ErrArgumentTokenMissing
	}

	return &Credential{
		accessToken: accessToken,
	}, nil
}

// AccessToken はアクセストークンを返します。
func (c *Credential) AccessToken() string {
	return c.accessToken
}
