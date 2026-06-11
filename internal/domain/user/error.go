package user

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid             = xerrors.Wrap(apperror.ErrValidation, "invalid user")
	ErrInvalidID           = xerrors.Wrap(errInvalid, "id failed")
	ErrInvalidFirstName    = xerrors.Wrap(errInvalid, "first name failed")
	ErrInvalidLastName     = xerrors.Wrap(errInvalid, "last name failed")
	ErrInvalidPasswordHash = xerrors.Wrap(errInvalid, "password hash failed")
	ErrInvalidEmail        = xerrors.Wrap(errInvalid, "email failed")
	ErrInvalidPhone        = xerrors.Wrap(errInvalid, "phone failed")
	ErrInvalidPrefectureID = xerrors.Wrap(errInvalid, "prefecture id failed")
	ErrInvalidCity         = xerrors.Wrap(errInvalid, "city failed")
	ErrInvalidStreet       = xerrors.Wrap(errInvalid, "street failed")
	ErrInvalidBuilding     = xerrors.Wrap(errInvalid, "building failed")
	ErrInvalidPostalCode   = xerrors.Wrap(errInvalid, "postal code failed")
	ErrInvalidUpdatedAt    = xerrors.Wrap(errInvalid, "updated at failed")
	ErrInvalidDeletedAt    = xerrors.Wrap(errInvalid, "deleted at failed")

	// ErrInvalidRawPassword は、RawPassword 値オブジェクト固有の検証エラーです。
	// User フィールド検証群（errInvalid 系）とは別系統で、errInvalid を経由しません。
	ErrInvalidRawPassword = xerrors.Wrap(apperror.ErrValidation, "invalid raw password")

	// ErrAlreadyDeleted は、既に論理削除済みのユーザーを再度削除しようとした場合のエラーです。
	ErrAlreadyDeleted = xerrors.Wrap(apperror.ErrConflict, "user is already deleted")

	// ErrCurrentPasswordMismatch は、パスワード変更時に現在のパスワードが一致しない場合のエラーです（422）。
	// authn 失敗（401）でも権限不足（403）でもなく、整形済みリクエストの意味的検証失敗として ErrValidation を用いる。
	ErrCurrentPasswordMismatch = xerrors.Wrap(apperror.ErrValidation, "current password does not match")
)
