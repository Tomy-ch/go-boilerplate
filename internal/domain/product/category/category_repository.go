//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package category

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// Repository は、商品カテゴリの永続化操作を定義するドメインリポジトリインターフェースです。
type Repository interface {
	// FindAll は、全商品カテゴリを sortKey 昇順で取得します。
	FindAll(ctx context.Context) (Categories, error)
	// FindByID は、ID から単一の商品カテゴリを取得します。未存在は NotFound を返します。
	FindByID(ctx context.Context, id uuid.UUID) (*Category, error)
}
