package inquiry

import (
	"fmt"

	"go-boilerplate/pkg/xerrors"
)

const (
	// AuthorKindUser は、送り手が利用者であることを表します。
	AuthorKindUser AuthorKind = "user"
	// AuthorKindOperator は、送り手が回答者（運営）であることを表します。
	AuthorKindOperator AuthorKind = "operator"
)

// AuthorKind は、メッセージの送り手の種別を表す値オブジェクトです。
type AuthorKind string

// NewAuthorKind は、種別を生成します。既知の 2 値以外は ErrInvalidAuthorKind を返します。
func NewAuthorKind(value string) (AuthorKind, error) {
	kind := AuthorKind(value)
	if !kind.valid() {
		return "", xerrors.Wrap(ErrInvalidAuthorKind, fmt.Sprintf("unknown author kind: %q", value))
	}
	return kind, nil
}

// String は、種別の文字列表現を返します。
func (k AuthorKind) String() string { return string(k) }

// valid は、既知の種別かどうかを返します。
func (k AuthorKind) valid() bool {
	return k == AuthorKindUser || k == AuthorKindOperator
}
