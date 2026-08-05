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
	// 退会可否の判定より前に取ることで、LockActiveShareByID の共有ロックと直列化します。
	// 論理削除済み・不存在はいずれも NotFound を返します。
	LockByID(ctx context.Context, id uuid.UUID) (*User, error)
	// LockActiveShareByID は、未削除ユーザーの在籍を共有ロックを取りながら確認します。共有ロック同士は
	// 両立するため同一ユーザーへの並行確認は直列化されず、LockByID の排他ロックとだけ衝突します。
	// 在籍の確認のみを行うためエンティティは返しません。
	// 論理削除済み・不存在はいずれも NotFound を返します。
	LockActiveShareByID(ctx context.Context, id uuid.UUID) error
}
