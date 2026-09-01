package inquirymessage

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid = xerrors.Wrap(apperror.ErrValidation, "invalid inquiry message")
	// ErrInvalidID は、メッセージ ID の検証に失敗した場合のエラーです（422）。
	ErrInvalidID = xerrors.Wrap(errInvalid, "id failed")
	// ErrInvalidInquiryID は、所属する問い合わせの ID の検証に失敗した場合のエラーです（422）。
	ErrInvalidInquiryID = xerrors.Wrap(errInvalid, "inquiryID failed")
	// ErrEmptyBody は、本文が空の場合のエラーです（422）。
	ErrEmptyBody = xerrors.Wrap(errInvalid, "body is empty")
	// ErrBodyTooLong は、本文が上限文字数を超える場合のエラーです（422）。
	ErrBodyTooLong = xerrors.Wrap(errInvalid, "body is too long")
	// ErrInvalidSequence は、問い合わせ内の位置が正の整数でない場合のエラーです（422）。
	ErrInvalidSequence = xerrors.Wrap(errInvalid, "sequence failed")
	// ErrInvalidAuthorKind は、送り手の種別が既知の 2 値でない場合のエラーです（422）。
	ErrInvalidAuthorKind = xerrors.Wrap(errInvalid, "author kind failed")
	// ErrInvalidAuthorSubject は、送り手の主体 ID の検証に失敗した場合のエラーです（422）。
	ErrInvalidAuthorSubject = xerrors.Wrap(errInvalid, "author subject failed")
)
