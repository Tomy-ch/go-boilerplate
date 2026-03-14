package user

import (
	"fmt"

	"boilerplate-go/pkg/stringkit"
	"boilerplate-go/pkg/xerrors"
)

type RawPassword struct {
	value string
}

// NewRawPassword は、パスワードの検証と生成を行います。
func NewRawPassword(v string) (RawPassword, error) {
	if !stringkit.InRange(v, MinRawPasswordLength, MaxRawPasswordLength) {
		return RawPassword{}, xerrors.Wrap(ErrPassword, fmt.Sprintf("password must be between %d and %d characters", MinRawPasswordLength, MaxRawPasswordLength))
	}
	return RawPassword{value: v}, nil
}

func (p RawPassword) Value() string {
	return p.value
}
