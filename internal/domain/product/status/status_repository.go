//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package status

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// Repository は、商品ステータスの永続化操作を定義するドメインリポジトリインターフェースです。
type Repository interface {
	// FindAll は、全商品ステータスを sortKey 昇順で取得します。
	FindAll(ctx context.Context) (Statuses, error)
	// FindByID は、ID から単一の商品ステータスを取得します。未存在は NotFound を返します。
	FindByID(ctx context.Context, id uuid.UUID) (*Status, error)
}
