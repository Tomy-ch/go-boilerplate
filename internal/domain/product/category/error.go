package category

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid = xerrors.Wrap(apperror.ErrValidation, "invalid product category")
	// ErrInvalidID は、商品カテゴリ ID の検証に失敗した場合のエラーです。
	ErrInvalidID = xerrors.Wrap(errInvalid, "id failed")
	// ErrInvalidName は、商品カテゴリ名の検証に失敗した場合のエラーです。
	ErrInvalidName = xerrors.Wrap(errInvalid, "name failed")
	// ErrInvalidCode は、商品カテゴリコードの検証に失敗した場合のエラーです。
	ErrInvalidCode = xerrors.Wrap(errInvalid, "code failed")
	// ErrInvalidSortKey は、商品カテゴリ表示順の検証に失敗した場合のエラーです。
	ErrInvalidSortKey = xerrors.Wrap(errInvalid, "sort key failed")
)
