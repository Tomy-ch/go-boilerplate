// Package safecast は、型変換の安全な関数を提供します。
package safecast

import (
	"fmt"
	"math"
)

// UintToInt は、uintをintに安全に変換します。
// オーバーフローが発生する場合はエラーを返します。
func UintToInt(x uint) (int, error) {
	const maxInt = math.MaxInt
	if x > maxInt {
		return 0, fmt.Errorf(
			"overflow: %d > MaxInt(%d): %w", x, maxInt, ErrOverflow,
		)
	}
	return int(x), nil
}
