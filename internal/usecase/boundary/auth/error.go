package auth

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	ErrUnauthorizedSubjectMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthorized: subject missing")
	ErrInvalidIDMissing           = xerrors.Wrap(apperror.ErrValidation, "invalid id: missing")
	ErrArgumentTokenMissing       = xerrors.Wrap(apperror.ErrInvalidArgument, "argument token missing")
)
