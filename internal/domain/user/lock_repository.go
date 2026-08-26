//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package user

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// LockRepository は、ユーザーの悲観ロックを伴う取得操作を定義するドメインリポジトリインターフェースです。
// 退会と、退会済みユーザーを拒む他集約の操作とを直列化する用途に限り、Repository から分けて定義します。
type LockRepository interface {
	// LockByID は、未削除の単一ユーザーを悲観ロック（排他）して取得します。退会は、このロックを
	// 退会可否の判定より前に取ることで、LockShareByID の共有ロックと直列化します。
	// 論理削除済み・不存在はいずれも NotFound を返します。
	LockByID(ctx context.Context, id uuid.UUID) (*User, error)
	// LockShareByID は、単一ユーザーを悲観ロック（共有）して取得します。共有ロック同士は両立するため
	// 同一ユーザーへの並行取得は直列化されず、LockByID の排他ロックとだけ衝突します。
	// 在籍しているかどうかは判定せず、退会済みのユーザーもそのまま返します（判定は IsActive の責務）。
	// 不存在は NotFound を返します。
	LockShareByID(ctx context.Context, id uuid.UUID) (*User, error)
}
