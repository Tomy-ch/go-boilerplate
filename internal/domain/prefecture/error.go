package prefecture

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	errInvalid               = xerrors.Wrap(apperror.ErrValidation, "invalid prefecture")
	ErrInvalidID             = xerrors.Wrap(errInvalid, "id failed")
	ErrInvalidPrefectureName = xerrors.Wrap(errInvalid, "prefecture name failed")
	ErrInvalidCode           = xerrors.Wrap(errInvalid, "code failed")
)
