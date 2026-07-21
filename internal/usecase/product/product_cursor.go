package product

import (
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// productCursorKeyCount は、商品一覧カーソルが保持するソートキーの個数（published_at, id）です。
const productCursorKeyCount = 2

// productCursor は、公開商品一覧（keyset ページネーション）の境界キーを表す usecase 層の値です。
// 直前ページ末尾行の公開日時と ID を保持し、次ページ取得時の keyset 比較の境界として用います。
// ドメイン層は不透明カーソルを持たず、この境界を primitive（published_at, id）で受け取ります。
type productCursor struct {
	publishedAt time.Time
	id          uuid.UUID
}

// decodeProductCursor は、cursor の不透明キー列を keyset 境界（productCursor）へ解釈します。
// 先頭ページ（カーソル無し）の場合は nil を返します。キーの個数・型が不正な場合は ErrInvalidArgument を返します。
func decodeProductCursor(cursor *paging.Cursor) (*productCursor, error) {
	if !cursor.HasCursor() {
		return nil, nil //nolint:nilnil // 先頭ページは境界なし（nil）を正常値として返す
	}

	keys := cursor.Keys()
	if len(keys) != productCursorKeyCount {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: expected 2 keys")
	}

	publishedAt, err := time.Parse(time.RFC3339Nano, keys[0])
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: published_at is not RFC3339Nano")
	}

	id, err := uuid.Parse(keys[1])
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: id is not a valid UUID")
	}

	return &productCursor{publishedAt: publishedAt, id: id}, nil
}

// encodeProductCursor は、現在ページ末尾行のソートキー（published_at, id）から次ページ用の不透明カーソルを生成します。
func encodeProductCursor(last *product.Product) string {
	return paging.EncodeCursor(last.PublishedAt().Format(time.RFC3339Nano), last.ID().String())
}
