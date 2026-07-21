package productstatus

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid = xerrors.Wrap(apperror.ErrValidation, "invalid product status")
	// ErrInvalidID は、商品ステータス ID の検証に失敗した場合のエラーです。
	ErrInvalidID = xerrors.Wrap(errInvalid, "id failed")
	// ErrInvalidName は、商品ステータス名の検証に失敗した場合のエラーです。
	ErrInvalidName = xerrors.Wrap(errInvalid, "name failed")
	// ErrInvalidCode は、商品ステータスコードの検証に失敗した場合のエラーです。
	ErrInvalidCode = xerrors.Wrap(errInvalid, "code failed")
	// ErrInvalidSortKey は、商品ステータスの表示順の検証に失敗した場合のエラーです。
	ErrInvalidSortKey = xerrors.Wrap(errInvalid, "sort key failed")
)
