//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE
package user

import (
	"context"
)

type Repository interface {
	// FindAll は、全ユーザーの情報ページング付きで取得します。
	FindAll(ctx context.Context, limit, offset int32) (Users, error)
	// Create は、ユーザーを作成します。
	Create(ctx context.Context, user *User) error
	// CountByActive は、アクティブ状態に基づいてユーザーの総件数を返します。
	CountByActive(ctx context.Context, active *bool) (int64, error)
}
