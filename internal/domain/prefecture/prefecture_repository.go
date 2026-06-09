//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE
package prefecture

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

type Repository interface {
	// FindByID は、IDから都道府県を取得します。存在しない場合は NotFound を返します。
	FindByID(ctx context.Context, id uuid.UUID) (*Prefecture, error)
	// FindByIDs は、複数IDから都道府県一覧を取得します。見つからないIDは結果から除外した部分一覧を返します（NotFound は返しません）。
	FindByIDs(ctx context.Context, ids []uuid.UUID) (Prefectures, error)
	// FindByName は、都道府県名から都道府県を取得します。存在しない場合は NotFound を返します。
	FindByName(ctx context.Context, name string) (*Prefecture, error)
}
