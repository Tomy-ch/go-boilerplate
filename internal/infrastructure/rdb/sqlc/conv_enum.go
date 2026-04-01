// Package sqlc は sqlc によって生成された値などを用いて列挙型の変換などを行うユーティリティを提供します。
package sqlc

import "boilerplate-go/internal/infrastructure/rdb/sqlc/gen"

// BoolToActiveState は、bool 型の値を gen.ActiveState 型に変換します。
//
//	引数が true の場合は gen.ActiveStateDeleted を返します。
//	引数が false の場合は gen.ActiveStateActive を返します。
func BoolToActiveState(b bool) gen.ActiveState {
	if b {
		return gen.ActiveStateActive
	}
	return gen.ActiveStateDeleted
}
