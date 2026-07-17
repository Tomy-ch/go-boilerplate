package user

import (
	"regexp"

	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/xerrors"
)

// emailPattern は、local@domain 形式（ドメインにドットを1つ以上含む）を要求します。
// 完全な RFC 5322 検証は wire 層（OpenAPI format: email）の責務であり、ドメイン層では
// 明らかな形式不正（@ 欠落・空白混入など）を弾く実用的な検証に留めます。
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Email は、検証済みのメールアドレスを保持する値オブジェクトです。
type Email struct {
	value string
}

// NewEmail は、与えられた文字列の長さと形式を検証し、有効な場合に Email を構築します。
// 違反時は ErrInvalidEmail を返します。
func NewEmail(v string) (Email, error) {
	if ok, msg := stringkit.ValidateInRange(v, minLength, maxEmailLength); !ok {
		return Email{}, xerrors.Wrap(ErrInvalidEmail, msg)
	}
	if !emailPattern.MatchString(v) {
		return Email{}, xerrors.Wrap(ErrInvalidEmail, "must be in local@domain format")
	}
	return Email{value: v}, nil
}

// Value は、保持しているメールアドレス文字列を返します。
func (e Email) Value() string {
	return e.value
}
