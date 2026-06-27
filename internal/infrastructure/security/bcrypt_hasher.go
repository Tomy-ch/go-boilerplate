// Package security は、パスワードの bcrypt ハッシュ化および照合を行う Hasher の実装を提供します。
package security

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/usecase/boundary/security"
	"go-boilerplate/pkg/xerrors"

	"golang.org/x/crypto/bcrypt"
)

type bcrypter struct {
	cost int
}

// NewBcryptHasher は、BcryptHasherを生成します。
func NewBcryptHasher(secCfg *config.SecurityConfig) security.Hasher {
	return &bcrypter{cost: secCfg.BcryptCost()}
}

// Hash は、パスワードをハッシュ化します。ハッシュ化に失敗した場合はエラーを返します。
func (b *bcrypter) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", xerrors.Wrap(apperror.ErrInternal, "bcrypt hash failed: "+err.Error())
	}
	return string(hash), nil
}

// Compare は、ハッシュとパスワードを照合します。一致すれば (true, nil)、不一致であれば (false, nil) を返します。ハッシュ形式不正などの技術的エラー時のみ (false, error) を返します。
func (b *bcrypter) Compare(hash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if xerrors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, xerrors.Wrap(apperror.ErrInternal, "bcrypt compare failed: "+err.Error())
	}
	return true, nil
}
