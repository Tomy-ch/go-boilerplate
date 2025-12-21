package prefecture

import (
	"boilerplate-go/internal/apperror"
	"boilerplate-go/pkg/xerrors"
)

var (
	errInvalid               = xerrors.Wrap(apperror.ErrValidation, "invalid prefecture")
	ErrInvalidID             = xerrors.Wrap(errInvalid, "id failed")
	ErrInvalidPrefectureName = xerrors.Wrap(errInvalid, "prefecture name failed")
	ErrInvalidCode           = xerrors.Wrap(errInvalid, "code failed")
)
