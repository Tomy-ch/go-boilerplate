// Package sqlc は sqlc によって生成された値などを用いて列挙型の変換などを行うユーティリティを提供します。
package sqlc

import "boilerplate-go/internal/infrastructure/rdb/sqlc/gen"

// BoolPtrToDeletedState は、*bool 型の値を gen.DeletedState 型に変換します。
//
//	引数が nil の場合は gen.DeletedStateAll を返します。
//	引数が true の場合は gen.DeletedStateDeleted を返します。
//	引数が false の場合は gen.DeletedStateActive を返します。
func BoolPtrToDeletedState(b *bool) gen.DeletedState {
	if b == nil {
		return gen.DeletedStateAll
	}
	if *b {
		return gen.DeletedStateDeleted
	}
	return gen.DeletedStateActive
}

// BoolToDeletedState は、bool 型の値を gen.DeletedState 型に変換します。
//
//	引数が true の場合は gen.DeletedStateDeleted を返します。
//	引数が false の場合は gen.DeletedStateActive を返します。
func BoolToDeletedState(b bool) gen.DeletedState {
	if b {
		return gen.DeletedStateDeleted
	}
	return gen.DeletedStateActive
}
