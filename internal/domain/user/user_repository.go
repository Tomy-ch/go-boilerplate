//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package user

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// Repository は、ユーザーの永続化操作を定義するドメインリポジトリインターフェースです。
type Repository interface {
	// FindByActive は、ユーザーの情報を、ページング付きで取得します。
	// active=nil で全件（削除済み含む）、true でアクティブのみ、false で削除済みのみを対象とします。
	FindByActive(ctx context.Context, active *bool, limit, offset int32) (Users, error)
	// FindFeed は、未削除ユーザーを (created_at DESC, id DESC) の安定順で keyset ページネーション取得します。
	// after=nil の場合は先頭ページを返し、それ以外は after が表す境界より後ろ（より過去）の行を返します。
	FindFeed(ctx context.Context, after *FeedCursor, limit int32) (Users, error)
	// FindByID は、IDから単一ユーザーを取得します。存在しない場合は NotFound を返します。
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	// Create は、ユーザーを作成します。
	Create(ctx context.Context, user *User) error
	// Update は、ユーザーの mutable フィールドと updatedAt / deletedAt を更新します。
	// 更新対象が存在しない場合は NotFound を返します。
	Update(ctx context.Context, user *User) error
	// CountByActive は、ユーザーの総件数を返します。
	// active=nil で全件（削除済み含む）、true でアクティブのみ、false で削除済みのみを対象とします。
	CountByActive(ctx context.Context, active *bool) (int64, error)
}
