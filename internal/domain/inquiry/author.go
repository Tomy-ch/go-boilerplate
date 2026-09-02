package inquiry

import (
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Author は、メッセージの送り手を表す値オブジェクトです。
// 利用者のときは問い合わせを開始した利用者を、回答者のときは運営側の利用者を指します。
type Author struct {
	kind      AuthorKind
	subjectID uuid.UUID
}

// NewAuthor は、送り手を生成します。
// 種別が既知の 2 値でなければ ErrInvalidAuthorKind を、主体 ID が未設定なら
// ErrInvalidAuthorSubject を返します。
func NewAuthor(kind AuthorKind, subjectID uuid.UUID) (Author, error) {
	if err := validateAuthor(kind, subjectID); err != nil {
		return Author{}, err
	}
	return Author{kind: kind, subjectID: subjectID}, nil
}

// validateAuthor は、送り手が満たすべき条件を検証します。
func validateAuthor(kind AuthorKind, subjectID uuid.UUID) error {
	if !kind.valid() {
		return xerrors.Wrap(ErrInvalidAuthorKind, "author kind is required")
	}
	if subjectID.IsNil() {
		return xerrors.Wrap(ErrInvalidAuthorSubject, "author subjectID is required")
	}
	return nil
}

// Kind は、送り手の種別を返します。
func (a Author) Kind() AuthorKind { return a.kind }

// SubjectID は、送り手の主体 ID を返します。
func (a Author) SubjectID() uuid.UUID { return a.subjectID }
