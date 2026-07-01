package authz

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// ErrForbidden は、認可が拒否された場合に返すエラーです（HTTP 403 Forbidden）。
var ErrForbidden = xerrors.Wrap(apperror.ErrPermissionDenied, "forbidden")
