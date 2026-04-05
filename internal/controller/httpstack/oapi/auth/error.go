package auth

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	ErrInvalidAuthDefaultMode          = xerrors.Wrap(apperror.ErrInternal, "default auth policy not found")
	ErrUnauthorizedTokenMissing        = xerrors.Wrap(apperror.ErrUnauthenticated, "authorization token missing")
	ErrUnauthorizedInvalidToken        = xerrors.Wrap(apperror.ErrUnauthenticated, "invalid token")
	ErrUnauthorizedEchoContextNotFound = xerrors.Wrap(apperror.ErrConflict, "echo context not found in request context")
	ErrUnauthorizedTokenNotProvided    = xerrors.Wrap(apperror.ErrUnauthenticated, "authorization token not provided")
)
