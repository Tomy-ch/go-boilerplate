package user

import (
	"fmt"

	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/xerrors"
)

type RawPassword struct {
	value string
}

// NewRawPassword は、与えられた文字列を検証し、有効な場合に RawPassword を構築します。
func NewRawPassword(v string) (RawPassword, error) {
	if !stringkit.InRange(v, MinRawPasswordLength, MaxRawPasswordLength) {
		return RawPassword{}, xerrors.Wrap(
			ErrInvalidRawPassword,
			fmt.Sprintf("password must be between %d and %d characters", MinRawPasswordLength, MaxRawPasswordLength),
		)
	}
	return RawPassword{value: v}, nil
}

func (p RawPassword) Value() string {
	return p.value
}

// String は、平文パスワードが fmt 経由で露出しないよう秘匿した文字列を返します。
func (p RawPassword) String() string { return "[REDACTED]" }

// GoString は、%#v 経由でも平文パスワードが露出しないよう秘匿した文字列を返します。
func (p RawPassword) GoString() string { return "[REDACTED]" }
