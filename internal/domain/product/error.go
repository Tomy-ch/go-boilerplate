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
	// ErrInvalidImagePath は、商品画像の画像パスの検証に失敗した場合のエラーです。
	ErrInvalidImagePath = xerrors.Wrap(errInvalid, "imagePath failed")
	// ErrInvalidImageSortKey は、商品画像の表示順の検証に失敗した場合のエラーです。
	ErrInvalidImageSortKey = xerrors.Wrap(errInvalid, "imageSortKey failed")
	// ErrDuplicateImageSortKey は、同一商品内で商品画像の表示順が重複している場合のエラーです。
	ErrDuplicateImageSortKey = xerrors.Wrap(errInvalid, "duplicate imageSortKey")
	// ErrInvalidStatusID は、商品ステータス ID の検証に失敗した場合のエラーです。
	ErrInvalidStatusID = xerrors.Wrap(errInvalid, "statusID failed")
	// ErrInvalidStatusName は、商品ステータス名の検証に失敗した場合のエラーです。
	ErrInvalidStatusName = xerrors.Wrap(errInvalid, "statusName failed")
	// ErrInvalidCategoryID は、商品カテゴリ ID の検証に失敗した場合のエラーです。
	ErrInvalidCategoryID = xerrors.Wrap(errInvalid, "categoryID failed")
	// ErrInvalidCategoryName は、商品カテゴリ名の検証に失敗した場合のエラーです。
	ErrInvalidCategoryName = xerrors.Wrap(errInvalid, "categoryName failed")
	// ErrInvalidVersion は、楽観ロックのバージョンの検証に失敗した場合のエラーです。
	ErrInvalidVersion = xerrors.Wrap(errInvalid, "version failed")
	// ErrVersionConflict は、読み込み後に他者が更新しており、楽観ロックのバージョンが一致しない場合のエラーです。
	// 同じ内容の再送では解消しないため、呼び出し元は最新を取得し直したうえでやり直す必要があります。
	ErrVersionConflict = xerrors.Wrap(apperror.ErrConflict, "product version conflict")
)
