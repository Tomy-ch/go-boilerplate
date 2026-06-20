package user

import (
	"time"

	"go-boilerplate/pkg/uuid"
)

// FeedCursor は、ユーザーフィード（keyset ページネーション）の境界キーを表す値オブジェクトです。
//
//	(created_at DESC, id DESC) の安定ソートに対応し、直前ページ末尾行の
//	作成日時と ID を保持します。クエリ層はこの値を keyset 比較
//	（WHERE (created_at, id) < (:created_at, :id)）の境界として解釈します。
type FeedCursor struct {
	// CreatedAt は、直前ページ末尾行の作成日時です。
	CreatedAt time.Time
	// ID は、直前ページ末尾行のユーザー ID です（同一作成日時のタイブレーク用）。
	ID uuid.UUID
}
