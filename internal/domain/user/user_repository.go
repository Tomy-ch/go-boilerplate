//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE
package user

import (
	"context"
	"time"
)

type Repository interface {
	// FindAll は、全ユーザーの情報ページング付きで取得します。
	FindAll(ctx context.Context, limit, offset int32) (Entities, error)
	// FindByKeyword は、キーワード検索でユーザーの情報を取得します。
	FindByKeyword(ctx context.Context, keywords []string, active *bool, limit, offset int32) (Entities, error)
	// CreateUser は、ユーザーを作成します。
	CreateUser(ctx context.Context, datetime time.Time, user *Entity) error
}
