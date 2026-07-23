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
}
