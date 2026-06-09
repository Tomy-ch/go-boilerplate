package safecast

import "errors"

// ErrOverflow は、型変換時にオーバーフローが発生したことを示すセンチネルエラーです。
// 分類判定（errors.Is）用に語を短く保ち、具体的な値・上限は呼び出し側が付与します。
var ErrOverflow = errors.New("overflow")
