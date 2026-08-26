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

	// ErrInvalidRoleID は、ロール ID の検証に失敗した場合のエラーです。
	ErrInvalidRoleID = xerrors.Wrap(errInvalid, "role id failed")
	// ErrInvalidRoleName は、ロール名の検証に失敗した場合のエラーです。
	ErrInvalidRoleName = xerrors.Wrap(errInvalid, "role name failed")
	// ErrInvalidRoleCode は、ロールコードの検証に失敗した場合のエラーです。
	ErrInvalidRoleCode = xerrors.Wrap(errInvalid, "role code failed")
	// ErrInvalidDeletedAt は、削除日時の検証に失敗した場合のエラーです。
	ErrInvalidDeletedAt = xerrors.Wrap(errInvalid, "deleted at failed")

	// ErrAlreadyDeleted は、既に論理削除済みのユーザーを再度削除しようとした場合のエラーです。
	ErrAlreadyDeleted = xerrors.Wrap(apperror.ErrConflict, "user is already deleted")
)
