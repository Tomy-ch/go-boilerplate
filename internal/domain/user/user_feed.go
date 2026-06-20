package user

import (
	"time"

	"go-boilerplate/pkg/uuid"
)

// FeedCursor は、ユーザーフィード（keyset ページネーション）の境界キーを表す値オブジェクトです。
//
//	直前ページ末尾行の作成日時と ID を保持し、次ページ取得時の keyset 比較の境界として用います。
type FeedCursor struct {
	createdAt time.Time
	id        uuid.UUID
}

// NewFeedCursor は、作成日時と ID から FeedCursor を生成します。
func NewFeedCursor(createdAt time.Time, id uuid.UUID) FeedCursor {
	return FeedCursor{createdAt: createdAt, id: id}
}

// CreatedAt は、境界となる作成日時を返します。
func (c FeedCursor) CreatedAt() time.Time { return c.createdAt }

// ID は、境界となるユーザー ID を返します。
func (c FeedCursor) ID() uuid.UUID { return c.id }
