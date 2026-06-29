// Package safecast は、型変換の安全な関数を提供します。
package safecast

import (
	"fmt"
	"math"

	"go-boilerplate/pkg/xerrors"
)

// UintToInt は、uintをintに安全に変換します。
// オーバーフローが発生する場合はエラーを返します。
func UintToInt(x uint) (int, error) {
	const maxInt = math.MaxInt
	if x > maxInt {
		return 0, xerrors.Wrap(ErrOverflow, fmt.Sprintf("uint %d exceeds MaxInt(%d)", x, maxInt))
	}
	return int(x), nil
}
