package stream

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// ErrCursorMalformed は、client が提示した cursor が形式不正・負数・表現範囲外のときのエラーです。
var ErrCursorMalformed = xerrors.Wrap(apperror.ErrInvalidArgument, "stream cursor is malformed")
