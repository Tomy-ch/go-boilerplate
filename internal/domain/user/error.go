package user

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid = xerrors.Wrap(apperror.ErrValidation, "invalid user")
	// ErrInvalidID は、ユーザー ID の検証に失敗した場合のエラーです。
	ErrInvalidID = xerrors.Wrap(errInvalid, "id failed")
	// ErrInvalidFirstName は、名の検証に失敗した場合のエラーです。
	ErrInvalidFirstName = xerrors.Wrap(errInvalid, "first name failed")
	// ErrInvalidLastName は、姓の検証に失敗した場合のエラーです。
	ErrInvalidLastName = xerrors.Wrap(errInvalid, "last name failed")
	// ErrInvalidPasswordHash は、パスワードハッシュの検証に失敗した場合のエラーです。
	ErrInvalidPasswordHash = xerrors.Wrap(errInvalid, "password hash failed")
	// ErrInvalidEmail は、メールアドレスの検証に失敗した場合のエラーです。
	ErrInvalidEmail = xerrors.Wrap(errInvalid, "email failed")
	// ErrInvalidPhone は、電話番号の検証に失敗した場合のエラーです。
	ErrInvalidPhone = xerrors.Wrap(errInvalid, "phone failed")
	// ErrInvalidPrefectureID は、都道府県 ID の検証に失敗した場合のエラーです。
	ErrInvalidPrefectureID = xerrors.Wrap(errInvalid, "prefecture id failed")
	// ErrInvalidCity は、市区町村の検証に失敗した場合のエラーです。
	ErrInvalidCity = xerrors.Wrap(errInvalid, "city failed")
	// ErrInvalidStreet は、番地の検証に失敗した場合のエラーです。
	ErrInvalidStreet = xerrors.Wrap(errInvalid, "street failed")
	// ErrInvalidBuilding は、建物名の検証に失敗した場合のエラーです。
	ErrInvalidBuilding = xerrors.Wrap(errInvalid, "building failed")
	// ErrInvalidPostalCode は、郵便番号の検証に失敗した場合のエラーです。
	ErrInvalidPostalCode = xerrors.Wrap(errInvalid, "postal code failed")
	// ErrInvalidUpdatedAt は、更新日時の検証に失敗した場合のエラーです。
	ErrInvalidUpdatedAt = xerrors.Wrap(errInvalid, "updated at failed")
	// ErrInvalidDeletedAt は、削除日時の検証に失敗した場合のエラーです。
	ErrInvalidDeletedAt = xerrors.Wrap(errInvalid, "deleted at failed")

	// ErrInvalidRawPassword は、RawPassword 値オブジェクト固有の検証エラーです。
	// User フィールド検証群（errInvalid 系）とは別系統で、errInvalid を経由しません。
	ErrInvalidRawPassword = xerrors.Wrap(apperror.ErrValidation, "invalid raw password")

	// ErrAlreadyDeleted は、既に論理削除済みのユーザーを再度削除しようとした場合のエラーです。
	ErrAlreadyDeleted = xerrors.Wrap(apperror.ErrConflict, "user is already deleted")

	// ErrCurrentPasswordMismatch は、パスワード変更時に現在のパスワードが一致しない場合のエラーです（422）。
	// authn 失敗（401）でも権限不足（403）でもなく、整形済みリクエストの意味的検証失敗として ErrValidation を用いる。
	ErrCurrentPasswordMismatch = xerrors.Wrap(apperror.ErrValidation, "current password does not match")
)
