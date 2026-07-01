// Package ptr は、ポインタ操作に関するユーティリティを提供します。
package ptr

// To は、引数として渡された値のポインタを返します。
func To[T any](v T) *T { return new(v) }

// Copy は、ポインタの指す値をシャローコピーした新しいポインタを返します（nil 入力時は nil）。
// T が参照型フィールド（スライス・マップ・ポインタ等）を含む場合、その参照先は複製元と共有されます。
func Copy[T any](v *T) *T {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}

// Deref は、p が非 nil ならその指す値を、nil なら fallback を返します。
func Deref[T any](p *T, fallback T) T {
	if p != nil {
		return *p
	}
	return fallback
}
