package user

import (
	"regexp"

	"go-boilerplate/pkg/xerrors"
)

// postalCodePattern は、日本の郵便番号形式（NNN-NNNN）を要求します。
// OpenAPI のリクエストスキーマ（UserBaseInputRequest.postalCode の pattern）と一致させています。
var postalCodePattern = regexp.MustCompile(`^[0-9]{3}-[0-9]{4}$`)

// PostalCode は、検証済みの郵便番号を保持する値オブジェクトです。
type PostalCode struct {
	value string
}

// NewPostalCode は、与えられた文字列の形式を検証し、有効な場合に PostalCode を構築します。
// 違反時は ErrInvalidPostalCode を返します。
func NewPostalCode(v string) (PostalCode, error) {
	if !postalCodePattern.MatchString(v) {
		return PostalCode{}, xerrors.Wrap(ErrInvalidPostalCode, "must be in NNN-NNNN format")
	}
	return PostalCode{value: v}, nil
}

// Value は、保持している郵便番号文字列を返します。
func (p PostalCode) Value() string {
	return p.value
}
