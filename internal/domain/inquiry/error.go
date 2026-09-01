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
)
