package safecast

import "go-boilerplate/pkg/xerrors"

// ErrOverflow は、型変換時にオーバーフローが発生したことを示すエラーです。
var ErrOverflow = xerrors.New("overflow")
