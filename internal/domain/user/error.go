package user

import (
	"boilerplate-go/internal/apperror"
	"boilerplate-go/pkg/xerrors"
)

var (
	errInvalid               = xerrors.Wrap(apperror.ErrValidation, "invalid user")
	ErrInvalidID             = xerrors.Wrap(errInvalid, "id: ")
	ErrInvalidFirstName      = xerrors.Wrap(errInvalid, "first name: ")
	ErrInvalidLastName       = xerrors.Wrap(errInvalid, "last name: ")
	ErrInvalidPassword       = xerrors.Wrap(errInvalid, "password: ")
	ErrInvalidEmail          = xerrors.Wrap(errInvalid, "email: ")
	ErrInvalidPhone          = xerrors.Wrap(errInvalid, "phone: ")
	ErrInvalidPrefectureID   = xerrors.Wrap(errInvalid, "prefecture id: ")
	ErrInvalidPrefectureName = xerrors.Wrap(errInvalid, "prefecture name: ")
	ErrInvalidCity           = xerrors.Wrap(errInvalid, "city: ")
	ErrInvalidStreet         = xerrors.Wrap(errInvalid, "street: ")
	ErrInvalidBuilding       = xerrors.Wrap(errInvalid, "building: ")
	ErrInvalidPostalCode     = xerrors.Wrap(errInvalid, "postal code: ")
	ErrInvalidDeletedAt      = xerrors.Wrap(errInvalid, "deleted at: ")
)
