package prefecture

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid = xerrors.Wrap(apperror.ErrValidation, "invalid prefecture")
	// ErrInvalidID は、都道府県 ID の検証に失敗した場合のエラーです。
	ErrInvalidID = xerrors.Wrap(errInvalid, "id failed")
	// ErrInvalidName は、都道府県名の検証に失敗した場合のエラーです。
	ErrInvalidName = xerrors.Wrap(errInvalid, "name failed")
	// ErrInvalidCode は、都道府県コードの検証に失敗した場合のエラーです。
	ErrInvalidCode = xerrors.Wrap(errInvalid, "code failed")
)
