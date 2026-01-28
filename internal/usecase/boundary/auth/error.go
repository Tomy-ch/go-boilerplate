package auth

import (
	"boilerplate-go/internal/apperror"
	"boilerplate-go/pkg/xerrors"
)

var (
	ErrUnauthorizedSubjectMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthorised: subject missing")
	ErrInvalidIDMissing           = xerrors.Wrap(apperror.ErrValidation, "invalid id: missing")
	ErrArgumentTokenMissing       = xerrors.Wrap(apperror.ErrInvalidArgument, "argument token missing")
)
