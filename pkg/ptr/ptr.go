// Package ptr は、ポインタ操作に関するユーティリティを提供します。
package ptr

// To は、引数として渡された値のポインタを返します。
func To[T any](v T) *T { return &v }

// Copy は、引数として渡されたポインタをコピーして返します。
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
