//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package purchase

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// Repository は、購入の永続化操作を定義するドメインリポジトリインターフェースです。
type Repository interface {
	// FindByID は、ID から購入を明細込みで取得します。書き込み後のドメイン整合の再検証と
	// レスポンス組み立ての取得元に用います。存在しない場合は NotFound を返します。
	FindByID(ctx context.Context, id uuid.UUID) (*Purchase, error)
	// LockByID は、対象の購入のみを悲観ロックして明細込みで再構築し返します。
	// 支払いの状態遷移の競合（同一購入への並行支払い）をこのロックで直列化します。擬似決済は購入集約
	// のみを更新するため、複数集約の原子性を要する CommandService ではなく Repository が担います。
	// 存在しない場合は NotFound を返します。
	LockByID(ctx context.Context, id uuid.UUID) (*Purchase, error)
	// UpdatePaid は、購入の状態遷移（→ 支払い済み）を、渡された ctx のトランザクション内で実行します。
	// 擬似決済のため購入集約のみを更新し、在庫操作は伴いません。対象は LockByID で取得・検証済みです。
	UpdatePaid(ctx context.Context, p *Purchase) error
	// UpdateShipped は、購入の状態遷移（→ 発送済み）を、渡された ctx のトランザクション内で実行します。
	// 配送追跡を扱わないため購入集約のみを更新し、在庫操作は伴いません。対象は LockByID で取得・検証済みです。
	UpdateShipped(ctx context.Context, p *Purchase) error
	// FindDetailByID は、ID から購入詳細（読み取りモデル）を明細込みで取得します。ステータス名は
	// 購入ステータスマスタで解決します（購入ステータスは購入集約に属する固定参照マスタのため、
	// 単一集約の Repository read です）。存在しない場合は NotFound を返します。
	FindDetailByID(ctx context.Context, id uuid.UUID) (*Detail, error)
	// FindFeedByUserID は、指定ユーザーの購入履歴を注文日時の降順（同時刻は ID 降順）の安定順で
	// keyset ページネーション取得します。ステータス名は購入ステータスマスタで解決します
	// （購入ステータスは購入集約に属する固定参照マスタのため、単一集約の Repository read です）。
	// params.AfterOrderedAt / AfterID が nil の場合は先頭ページを返します。
	FindFeedByUserID(ctx context.Context, userID uuid.UUID, params ListFeedParams) ([]FeedItem, error)
	// ExistsInProgressByUserID は、指定ユーザーに進行中の購入が 1 件でも存在するかを返します。
	// 進行中は TerminalStatusCodes のいずれでもないステータスの購入を指します。
	ExistsInProgressByUserID(ctx context.Context, userID uuid.UUID) (bool, error)
}
