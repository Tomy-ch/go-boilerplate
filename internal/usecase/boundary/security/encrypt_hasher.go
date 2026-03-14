//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package security は、セキュリティ関連のドメインを提供します。
package security

// Encrypter は、パスワードをハッシュ化および比較するためのインターフェースです。
type Encrypter interface {
	// Hash は、パスワードをハッシュ化します。ハッシュ化に失敗した場合はエラーを返します。
	Hash(password string) (string, error)
	// Compare は、ハッシュとパスワードを比較します。比較に失敗した場合はエラーを返します。
	Compare(hash, password string) (bool, error)
}
