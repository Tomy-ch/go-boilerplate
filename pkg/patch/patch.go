// Package patch は、部分更新の入力が取りうる状態を表す値を提供します。
package patch

// Field は、部分更新における 1 フィールドの指定状態を表します。
// 未指定（現在値を据え置く）・null 指定（値をクリアする）・値指定（その値へ更新する）の 3 状態を区別します。
// ゼロ値は未指定です。
type Field[T any] struct {
	specified bool
	value     *T
}

// Unspecified は、未指定（現在値を据え置く）のフィールドを返します。
func Unspecified[T any]() Field[T] { return Field[T]{} }

// Null は、null 指定（値をクリアする）のフィールドを返します。
func Null[T any]() Field[T] { return Field[T]{specified: true} }

// Value は、値指定（v へ更新する）のフィールドを返します。
func Value[T any](v T) Field[T] { return Field[T]{specified: true, value: &v} }

// Resolve は、現在値 current へ部分更新を適用した結果を返します。
// 未指定なら current をそのまま、null 指定なら nil、値指定ならその値の新しいポインタを返します。
func (f Field[T]) Resolve(current *T) *T {
	if !f.specified {
		return current
	}
	if f.value == nil {
		return nil
	}
	v := *f.value
	return &v
}
