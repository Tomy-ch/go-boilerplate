// Package security は、セキュリティ関連のインフラストラクチャを提供します。
package security

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/usecase/boundary/security"
	"go-boilerplate/pkg/xerrors"

	"golang.org/x/crypto/bcrypt"
)

type bcrypter struct {
	cost int
}

// NewBcryptHasher は、BcryptHasherを生成します。
func NewBcryptHasher(secCfg *config.SecurityConfig) security.Encrypter {
	return &bcrypter{cost: secCfg.BcryptCost()}
}

// Hash は、パスワードをハッシュ化します。ハッシュ化に失敗した場合はエラーを返します。
func (b *bcrypter) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Compare は、ハッシュとパスワードを比較します。比較に失敗した場合はエラーを返します。
func (b *bcrypter) Compare(hash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if xerrors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
