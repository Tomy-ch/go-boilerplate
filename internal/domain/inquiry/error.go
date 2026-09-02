package inquiry

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid = xerrors.Wrap(apperror.ErrValidation, "invalid inquiry")
	// ErrInvalidID は、問い合わせ ID の検証に失敗した場合のエラーです（422）。
	ErrInvalidID = xerrors.Wrap(errInvalid, "id failed")
	// ErrInvalidUserID は、問い合わせを開始した利用者の ID の検証に失敗した場合のエラーです（422）。
	ErrInvalidUserID = xerrors.Wrap(errInvalid, "userID failed")
	// ErrInvalidTime は、時刻の前後関係が満たされない場合のエラーです（422）。
	ErrInvalidTime = xerrors.Wrap(errInvalid, "time ordering failed")

	errInvalidMessage = xerrors.Wrap(apperror.ErrValidation, "invalid inquiry message")
	// ErrInvalidMessageID は、メッセージ ID の検証に失敗した場合のエラーです（422）。
	ErrInvalidMessageID = xerrors.Wrap(errInvalidMessage, "id failed")
	// ErrEmptyBody は、メッセージ本文が空の場合のエラーです（422）。
	ErrEmptyBody = xerrors.Wrap(errInvalidMessage, "body is empty")
	// ErrBodyTooLong は、メッセージ本文が上限文字数を超える場合のエラーです（422）。
	ErrBodyTooLong = xerrors.Wrap(errInvalidMessage, "body is too long")
	// ErrInvalidSequence は、問い合わせ内の位置が正の整数でない場合のエラーです（422）。
	ErrInvalidSequence = xerrors.Wrap(errInvalidMessage, "sequence failed")
	// ErrInvalidAuthorKind は、送り手の種別が既知の 2 値でない場合のエラーです（422）。
	ErrInvalidAuthorKind = xerrors.Wrap(errInvalidMessage, "author kind failed")
	// ErrInvalidAuthorSubject は、送り手の主体 ID の検証に失敗した場合のエラーです（422）。
	ErrInvalidAuthorSubject = xerrors.Wrap(errInvalidMessage, "author subject failed")
)
