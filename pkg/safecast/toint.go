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

// IntToInt32 は、intをint32に安全に変換します。
// int32 の範囲外の場合はエラーを返します。
func IntToInt32(x int) (int32, error) {
	if x < math.MinInt32 || x > math.MaxInt32 {
		return 0, xerrors.Wrap(ErrOverflow, fmt.Sprintf("int %d exceeds int32 range [%d,%d]", x, math.MinInt32, math.MaxInt32))
	}
	return int32(x), nil
}

// IntToInt16 は、intをint16に安全に変換します。
// int16 の範囲外の場合はエラーを返します。
func IntToInt16(x int) (int16, error) {
	if x < math.MinInt16 || x > math.MaxInt16 {
		return 0, xerrors.Wrap(ErrOverflow, fmt.Sprintf("int %d exceeds int16 range [%d,%d]", x, math.MinInt16, math.MaxInt16))
	}
	return int16(x), nil
}

// IntPtrToInt32Ptr は、任意指定の*intを*int32に安全に変換します。
// nil は変換対象なしとして nil を返し、int32 の範囲外の場合はエラーを返します。
func IntPtrToInt32Ptr(x *int) (*int32, error) {
	if x == nil {
		return nil, nil //nolint:nilnil // 未設定は変換対象なしを表すため nil, nil が正常値
	}

	v, err := IntToInt32(*x)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
