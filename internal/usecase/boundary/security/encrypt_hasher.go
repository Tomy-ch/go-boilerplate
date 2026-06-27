//go:generate mockgen -source=$GOFILE -destination=mock/mock_encrypt_hasher.gen.go -package=mock_$GOPACKAGE

// Package security は、ユースケースが必要とするパスワードのハッシュ化・比較機能のインターフェース（境界）を提供します。
package security

// Hasher は、パスワードをハッシュ化および比較するためのインターフェースです。
type Hasher interface {
	// Hash は、パスワードをハッシュ化します。ハッシュ化に失敗した場合はエラーを返します。
	Hash(password string) (string, error)
	// Compare は、hash とパスワードが一致するかを検証します。一致すれば (true, nil)、不一致なら (false, nil) を返します。ハッシュ形式が不正など系レベルの失敗時のみ error を返します。
	Compare(hash, password string) (bool, error)
}
