package user

import (
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// feedCursorKeyCount は、フィードカーソルが保持するソートキーの個数（created_at, id）です。
const feedCursorKeyCount = 2

// decodeFeedCursor は、cursor の不透明キー列を keyset 境界（FeedCursor）へ解釈します。
// 先頭ページ（カーソル無し）の場合は nil を返します。キーの個数・型が不正な場合は ErrInvalidArgument を返します。
func decodeFeedCursor(cursor *paging.Cursor) (*user.FeedCursor, error) {
	if !cursor.HasCursor() {
		return nil, nil //nolint:nilnil // 先頭ページは境界なし（nil）を正常値として返す
	}

	keys := cursor.Keys()
	if len(keys) != feedCursorKeyCount {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: expected 2 keys")
	}

	createdAt, err := time.Parse(time.RFC3339Nano, keys[0])
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: created_at is not RFC3339Nano")
	}

	id, err := uuid.Parse(keys[1])
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: id is not a valid UUID")
	}

	fc := user.NewFeedCursor(createdAt, id)
	return &fc, nil
}

// encodeFeedCursor は、現在ページ末尾行のソートキー（created_at, id）から次ページ用の不透明カーソルを生成します。
func encodeFeedCursor(last *user.User) string {
	return paging.EncodeCursor(last.CreatedAt().Format(time.RFC3339Nano), last.ID().String())
}
