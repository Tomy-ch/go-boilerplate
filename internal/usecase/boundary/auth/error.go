package auth

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	ErrUnauthenticatedSubjectMissing = xerrors.Wrap(apperror.ErrUnauthenticated, "unauthenticated: subject missing")
	ErrSubjectNotUUID                = xerrors.Wrap(apperror.ErrValidation, "id unavailable: subject is not a uuid")
	ErrTokenMissing                  = xerrors.Wrap(apperror.ErrInvalidArgument, "token missing")
)
