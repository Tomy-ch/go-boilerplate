//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package purchase

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// Repository は、購入の永続化操作を定義するドメインリポジトリインターフェースです。
type Repository interface {
	// FindByID は、ID から購入を明細込みで取得します。存在しない場合は NotFound を返します。
	FindByID(ctx context.Context, id uuid.UUID) (*Purchase, error)
	// LockByCode は、購入コードから対象の購入のみを悲観ロックして明細込みで再構築し返します。
	// 状態遷移の競合（同一購入への並行更新）をこのロックで直列化します。
	// 存在しない場合は NotFound を返します。
	LockByCode(ctx context.Context, code string) (*Purchase, error)
	// UpdatePaid は、購入の状態遷移（→ 支払い済み）を、渡された ctx のトランザクション内で実行します。
	// 対象は LockByCode で取得・検証済みです。
	UpdatePaid(ctx context.Context, p *Purchase) error
	// UpdateShipped は、購入の状態遷移（→ 発送済み）を、渡された ctx のトランザクション内で実行します。
	// 対象は LockByCode で取得・検証済みです。
	UpdateShipped(ctx context.Context, p *Purchase) error
	// UpdateDelivered は、購入の状態遷移（→ 配達済み）を、渡された ctx のトランザクション内で実行します。
	// 対象は LockByCode で取得・検証済みです。
	UpdateDelivered(ctx context.Context, p *Purchase) error
	// FindShippable は、発送可能な購入を注文日時の古い順（同時刻は ID 昇順）で明細込みに最大 limit 件
	// 取得します。該当が無い場合は空を返します。絞り込みは Purchase.IsShippable の定義と一致させること。
	FindShippable(ctx context.Context, limit int32) (Purchases, error)
	// FindDetailByID は、ID から購入詳細（読み取りモデル）を明細込みで取得します。ステータス名は
	// 購入ステータスマスタで解決します。存在しない場合は NotFound を返します。
	FindDetailByID(ctx context.Context, id uuid.UUID) (*Detail, error)
	// FindStatusesByUserID は、指定ユーザーの購入が取っているステータスを重複なく返します。
	// 進行中かどうかで絞り込まないため、その判定（Status.IsTerminal の否定）は呼び出し側が行います。
	// 購入を 1 件も持たない場合は空を返し、順序は保証しません。
	FindStatusesByUserID(ctx context.Context, userID uuid.UUID) ([]Status, error)
	// FindUserIDsWithPurchases は、与えたユーザー ID のうち、購入を 1 件以上持つものを返します。
	// ステータスは問わず、順序は保証しません。userIDs が空の場合は空を返します。
	FindUserIDsWithPurchases(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error)
}
