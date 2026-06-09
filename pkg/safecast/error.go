package safecast

import "errors"

// ErrOverflow は、型変換時にオーバーフローが発生したことを示すエラーです。
var ErrOverflow = errors.New("overflow")
