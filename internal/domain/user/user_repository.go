//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE
package user

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

type Repository interface {
	// FindByActive は、アクティブ状態に基づいてユーザーの情報ページング付きで取得します。
	FindByActive(ctx context.Context, active *bool, limit, offset int32) (Users, error)
	// FindByID は、IDから単一ユーザーを取得します。存在しない場合は NotFound を返します。
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	// Create は、ユーザーを作成します。
	Create(ctx context.Context, user *User) error
	// Update は、ユーザーの mutable フィールドと updatedAt / deletedAt を更新します。
	Update(ctx context.Context, user *User) error
	// CountByActive は、アクティブ状態に基づいてユーザーの総件数を返します。
	CountByActive(ctx context.Context, active *bool) (int64, error)
}
