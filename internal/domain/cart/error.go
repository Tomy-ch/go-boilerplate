package cart

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid = xerrors.Wrap(apperror.ErrValidation, "invalid cart")
	// ErrInvalidID は、カート ID または明細 ID の検証に失敗した場合のエラーです（422）。
	ErrInvalidID = xerrors.Wrap(errInvalid, "id failed")
	// ErrInvalidUserID は、所有者のユーザー ID の検証に失敗した場合のエラーです（422）。
	ErrInvalidUserID = xerrors.Wrap(errInvalid, "userID failed")
	// ErrInvalidProductID は、明細の商品 ID の検証に失敗した場合のエラーです（422）。
	ErrInvalidProductID = xerrors.Wrap(errInvalid, "productID failed")
	// ErrInvalidQuantity は、数量が許容範囲を外れた場合のエラーです（422）。
	ErrInvalidQuantity = xerrors.Wrap(errInvalid, "quantity failed")
	// ErrTooManyItems は、明細数が上限を超える場合のエラーです（422）。
	ErrTooManyItems = xerrors.Wrap(errInvalid, "too many items")
	// ErrDuplicateProductID は、明細に同一 productID が重複した場合のエラーです（422）。
	ErrDuplicateProductID = xerrors.Wrap(errInvalid, "duplicate product id in items")
	// ErrInvalidSessionToken は、セッショントークンの形式が不正な場合のエラーです（422）。
	ErrInvalidSessionToken = xerrors.Wrap(errInvalid, "session token failed")
	// ErrInvalidOwner は、所有者とセッショントークンの排他が満たされない場合のエラーです（422）。
	ErrInvalidOwner = xerrors.Wrap(errInvalid, "exactly one of ownerID and sessionToken must be set")
	// ErrInvalidExpiresAt は、有効期限の検証に失敗した場合のエラーです（422）。
	ErrInvalidExpiresAt = xerrors.Wrap(errInvalid, "expiresAt failed")

	// 以下は状態の衝突であってリクエストの不正ではないため、apperror.ErrConflict を基底に持ちます
	// （HTTP ステータスへの写像は internal/apperror/README.md の Mapping Table）。

	// ErrAlreadyOwned は、所有者が確定済みのカートに再度所有者を設定しようとした場合のエラーです（409）。
	ErrAlreadyOwned = xerrors.Wrap(apperror.ErrConflict, "cart is already owned")
)
