package user

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid               = xerrors.Wrap(apperror.ErrValidation, "invalid user")
	ErrInvalidID             = xerrors.Wrap(errInvalid, "id failed")
	ErrInvalidFirstName      = xerrors.Wrap(errInvalid, "first name failed")
	ErrInvalidLastName       = xerrors.Wrap(errInvalid, "last name failed")
	ErrInvalidPasswordHash   = xerrors.Wrap(errInvalid, "password hash failed")
	ErrInvalidEmail          = xerrors.Wrap(errInvalid, "email failed")
	ErrInvalidPhone          = xerrors.Wrap(errInvalid, "phone failed")
	ErrInvalidPrefectureID   = xerrors.Wrap(errInvalid, "prefecture id failed")
	ErrInvalidPrefectureName = xerrors.Wrap(errInvalid, "prefecture name failed")
	ErrInvalidCity           = xerrors.Wrap(errInvalid, "city failed")
	ErrInvalidStreet         = xerrors.Wrap(errInvalid, "street failed")
	ErrInvalidBuilding       = xerrors.Wrap(errInvalid, "building failed")
	ErrInvalidPostalCode     = xerrors.Wrap(errInvalid, "postal code failed")
	ErrInvalidUpdatedAt      = xerrors.Wrap(errInvalid, "updated at failed")
	ErrInvalidDeletedAt      = xerrors.Wrap(errInvalid, "deleted at failed")

	ErrInvalidRawPassword = xerrors.Wrap(apperror.ErrValidation, "invalid raw password")
)
