package cart

import (
	"fmt"

	"go-boilerplate/pkg/xerrors"
)

// SessionToken は、所有者が確定していないカートを追跡するための値オブジェクトです。
// 値の生成は外側の層が行い、ドメインは受け取った値の形式のみを検証します。
type SessionToken struct {
	value string
}

// NewSessionToken は、セッショントークンを生成します。
// 長さが規定と異なる場合、または URL-safe でない文字を含む場合は ErrInvalidSessionToken を返します。
func NewSessionToken(value string) (SessionToken, error) {
	if len(value) != sessionTokenLength {
		return SessionToken{}, xerrors.Wrap(
			ErrInvalidSessionToken,
			fmt.Sprintf("length must be %d, got %d", sessionTokenLength, len(value)),
		)
	}
	for _, r := range value {
		if !isURLSafe(r) {
			return SessionToken{}, xerrors.Wrap(
				ErrInvalidSessionToken, fmt.Sprintf("contains a character that is not URL-safe: %q", r),
			)
		}
	}
	return SessionToken{value: value}, nil
}

// isURLSafe は、base64url のアルファベット（RFC 4648 §5）に含まれる文字かどうかを返します。
func isURLSafe(r rune) bool {
	switch {
	case r >= 'A' && r <= 'Z':
		return true
	case r >= 'a' && r <= 'z':
		return true
	case r >= '0' && r <= '9':
		return true
	case r == '-' || r == '_':
		return true
	default:
		return false
	}
}

// IsZero は、生成を経ていないゼロ値かどうかを返します。
// 複合リテラルは検証を通らずに組み立てられるため、値を受け取る側はこれで拒否できます。
func (t SessionToken) IsZero() bool { return t.value == "" }

// Value は、セッショントークンの文字列表現を返します。
func (t SessionToken) Value() string { return t.value }
