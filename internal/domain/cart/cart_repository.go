//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package cart

import (
	"context"
	"time"

	"go-boilerplate/pkg/uuid"
)

// Repository は、カート集約の永続化を抽象化します。
//
// 明細はカートに属する子であり単独では存在しないため、明細だけを対象とする操作は持ちません。
// 書き込みは集約単位で、Create と Update の 2 つに閉じます。
type Repository interface {
	// FindBySessionToken は、セッショントークンからカートを明細込みで取得します。
	// 存在しない場合は NotFound を返します。所有者が確定したカートはトークンを持たないため、
	// このメソッドでは引けません。
	FindBySessionToken(ctx context.Context, token SessionToken) (*Cart, error)

	// FindByOwnerID は、所有者からカートを明細込みで取得します。
	// 存在しない場合は NotFound を返します。ユーザー 1 人につきカートは高々 1 件です。
	//
	// 期限切れのカートも返します。期限切れかどうかは Cart.IsExpired が定義するため、
	// ここで取り除くと呼び出し側から判定の機会が失われます。
	FindByOwnerID(ctx context.Context, userID uuid.UUID) (*Cart, error)

	// LockByID は、更新のためにカートを悲観ロックして明細込みで取得します。
	// ロックは同一カートに対する並行した更新を、取得から commit まで直列化します。
	// 2 件を同時にロックする場合は ID 昇順で取得し、ロック順序を全 tx で固定します
	// （ADR-0034 (ordered-pessimistic-row-locks)）。
	LockByID(ctx context.Context, id uuid.UUID) (*Cart, error)

	// Create は、カートを明細込みで新規登録します。
	// 所有者またはセッショントークンが既に使われている場合は Conflict を返します。
	Create(ctx context.Context, c *Cart) error

	// Update は、カートを明細込みで現在の状態へ一致させます（差分ではなく集約単位の書き込み）。
	// 所有者・セッショントークン・有効期限と、明細の集合が対象です。
	// 対象が存在しない場合は NotFound を返します。
	//
	// 明細は商品ごとに置き換えられ、最初に入った時刻は保持されます。集合から消えた明細は取り除かれます。
	Update(ctx context.Context, c *Cart) error

	// Delete は、カートを明細ごと削除します。存在しない場合もエラーとしません。
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteExpired は、有効期限を過ぎたカートを最大 limit 件削除し、削除件数を返します。
	// 1 回で消し切ることを意図せず、件数上限で区切って繰り返し呼ばれる前提です。
	//
	// 削除の対象は Cart.IsExpired が定義する「期限切れ」であり、この実装はその実行形です。
	// 片方だけを変更してはなりません。
	DeleteExpired(ctx context.Context, now time.Time, limit int32) (int, error)
}
