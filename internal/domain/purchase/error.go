package purchase

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid = xerrors.Wrap(apperror.ErrValidation, "invalid purchase")
	// ErrEmptyDetails は、明細が空の場合のエラーです（422）。
	ErrEmptyDetails = xerrors.Wrap(errInvalid, "details must not be empty")
	// ErrDuplicateProductID は、明細に同一 productID が重複した場合のエラーです（422）。
	// 在庫のロック順序を固定するため、重複は入力段階で弾きます。
	ErrDuplicateProductID = xerrors.Wrap(errInvalid, "duplicate product id in details")
	// ErrInvalidQuantity は、購入数量が最小値未満の場合のエラーです（422）。
	ErrInvalidQuantity = xerrors.Wrap(errInvalid, "quantity must be positive")
	// ErrProductNotFound は、明細の productID に対応する（ロック済みの）商品が存在しない場合のエラーです（422）。
	ErrProductNotFound = xerrors.Wrap(errInvalid, "product not found for detail")
	// ErrInvalidID は、購入 ID の検証に失敗した場合のエラーです。
	ErrInvalidID = xerrors.Wrap(errInvalid, "id failed")
	// ErrInvalidCode は、購入コードの検証に失敗した場合のエラーです。
	ErrInvalidCode = xerrors.Wrap(errInvalid, "code failed")
	// ErrInvalidUserID は、ユーザー ID の検証に失敗した場合のエラーです。
	ErrInvalidUserID = xerrors.Wrap(errInvalid, "userID failed")
	// ErrInvalidStatusID は、ステータス ID の検証に失敗した場合のエラーです（再構築時）。
	ErrInvalidStatusID = xerrors.Wrap(errInvalid, "statusID failed")
	// ErrInvalidAmount は、金額の検証に失敗した場合のエラーです（再構築時）。
	ErrInvalidAmount = xerrors.Wrap(errInvalid, "amount failed")

	// ErrInsufficientStock は、在庫不足（売り越し）の場合のエラーです。
	// 在庫は時間依存の外部状態でありリクエスト自体は妥当なため、409（ErrConflict）へ写像します（ADR-0039）。
	ErrInsufficientStock = xerrors.Wrap(apperror.ErrConflict, "insufficient stock")

	// ErrAlreadyCanceled は、既にキャンセル済みの購入を再度キャンセルしようとした場合のエラーです。
	// 冪等でない状態遷移の衝突であり、409（ErrConflict）へ写像します（ADR-0039）。
	ErrAlreadyCanceled = xerrors.Wrap(apperror.ErrConflict, "purchase already canceled")

	// ErrCancelNotAllowed は、キャンセル不可の状態（完了・発送済み・配達済み）からキャンセルしようとした場合のエラーです。
	// 状態機械上の不正遷移であり、409（ErrConflict）へ写像します（ADR-0039）。
	ErrCancelNotAllowed = xerrors.Wrap(apperror.ErrConflict, "purchase cannot be canceled in the current state")

	// ErrAlreadyPaid は、既に支払い済みの購入を再度支払おうとした場合のエラーです（二重支払い）。
	// 冪等でない状態遷移の衝突であり、409（ErrConflict）へ写像します（ADR-0039）。
	ErrAlreadyPaid = xerrors.Wrap(apperror.ErrConflict, "purchase already paid")

	// ErrPayNotAllowed は、支払い不可の状態（キャンセル済み・完了・発送済み・配達済み）から支払おうとした場合のエラーです。
	// 状態機械上の不正遷移であり、409（ErrConflict）へ写像します（ADR-0039）。
	ErrPayNotAllowed = xerrors.Wrap(apperror.ErrConflict, "purchase cannot be paid in the current state")
)
