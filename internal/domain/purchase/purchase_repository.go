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
	// FindFeedByUserID は、指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で
	// keyset ページネーション取得します。ステータス名は購入ステータスマスタとの結合で解決します
	// （購入ステータスは購入集約に属する固定参照マスタのため、単一集約の Repository read です）。
	// params.AfterOrderedAt / AfterID が nil の場合は先頭ページを返します。
	FindFeedByUserID(ctx context.Context, userID uuid.UUID, params ListFeedParams) ([]FeedItem, error)
}
