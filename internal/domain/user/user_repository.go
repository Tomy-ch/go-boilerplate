//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package user

import (
	"context"
	"time"

	"go-boilerplate/pkg/uuid"
)

// Repository は、ユーザーの永続化操作を定義するドメインリポジトリインターフェースです。
type Repository interface {
	// FindByActive は、ユーザーの情報を、ページング付きで取得します。
	// active=nil で全件（削除済み含む）、true でアクティブのみ、false で削除済みのみを対象とします。
	FindByActive(ctx context.Context, active *bool, limit, offset int32) (Users, error)
	// FindFeed は、未削除ユーザーを作成日時の降順（同時刻は ID 降順）の安定順で keyset ページネーション取得します。
	// after=nil の場合は先頭ページを返し、それ以外は after が表す境界より後ろ（より過去）を返します。
	FindFeed(ctx context.Context, after *FeedCursor, limit int32) (Users, error)
	// SearchByKeyword は、検索テキストがいずれかのキーワードに部分一致するユーザーを、作成日時の降順でページング取得します。
	// active=nil で全件（削除済み含む）、true でアクティブのみ、false で削除済みのみを対象とします。keywords が空の場合は全ユーザーを対象とします。
	SearchByKeyword(ctx context.Context, keywords []string, active *bool, limit, offset int32) (Users, error)
	// CountByKeyword は、検索テキストがいずれかのキーワードに部分一致するユーザーの総件数を返します。
	// active / keywords の意味は SearchByKeyword と同じです。
	CountByKeyword(ctx context.Context, keywords []string, active *bool) (int64, error)
	// FindByID は、IDから単一ユーザーを取得します。存在しない場合は NotFound を返します。
	// 悲観ロックを伴う取得は LockRepository が持ちます。
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	// Create は、ユーザーを作成します。
	Create(ctx context.Context, user *User) error
	// Update は、ユーザーの mutable フィールドと updatedAt / deletedAt を更新します。
	// 更新対象が存在しない場合は NotFound を返します。
	Update(ctx context.Context, user *User) error
	// CountByActive は、ユーザーの総件数を返します。
	// active=nil で全件（削除済み含む）、true でアクティブのみ、false で削除済みのみを対象とします。
	CountByActive(ctx context.Context, active *bool) (int64, error)
	// FindDeletedBefore は、cutoff より前に論理削除されたユーザーの ID を、ID の昇順で最大 limit 件返します。
	// afterID=nil の場合は先頭から、それ以外は afterID より後ろを返します。
	FindDeletedBefore(ctx context.Context, cutoff time.Time, afterID *uuid.UUID, limit int32) ([]uuid.UUID, error)
	// PurgeByIDs は、指定した ID のユーザーを従属データごと物理削除し、削除したユーザーの件数を返します。
	// 論理削除されていないユーザーは従属データを含めて削除されないため、返る件数が ids の件数を下回ることがあります。
	// ids が空の場合は何も削除せず 0 を返します。
	PurgeByIDs(ctx context.Context, ids []uuid.UUID) (int64, error)
}
