package product

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid = xerrors.Wrap(apperror.ErrValidation, "invalid product")
	// ErrInvalidID は、商品 ID の検証に失敗した場合のエラーです。
	ErrInvalidID = xerrors.Wrap(errInvalid, "id failed")
	// ErrInvalidName は、商品名の検証に失敗した場合のエラーです。
	ErrInvalidName = xerrors.Wrap(errInvalid, "name failed")
	// ErrInvalidQuantity は、在庫数の検証に失敗した場合のエラーです。
	ErrInvalidQuantity = xerrors.Wrap(errInvalid, "quantity failed")
	// ErrInvalidStockWarningThreshold は、在庫警告閾値の検証に失敗した場合のエラーです。
	ErrInvalidStockWarningThreshold = xerrors.Wrap(errInvalid, "stockWarningThreshold failed")
	// ErrInvalidStatusID は、商品ステータス ID の検証に失敗した場合のエラーです。
	ErrInvalidStatusID = xerrors.Wrap(errInvalid, "statusID failed")
	// ErrInvalidStatusName は、商品ステータス名の検証に失敗した場合のエラーです。
	ErrInvalidStatusName = xerrors.Wrap(errInvalid, "statusName failed")
	// ErrInvalidCategoryID は、商品カテゴリ ID の検証に失敗した場合のエラーです。
	ErrInvalidCategoryID = xerrors.Wrap(errInvalid, "categoryID failed")
	// ErrInvalidCategoryName は、商品カテゴリ名の検証に失敗した場合のエラーです。
	ErrInvalidCategoryName = xerrors.Wrap(errInvalid, "categoryName failed")
)
