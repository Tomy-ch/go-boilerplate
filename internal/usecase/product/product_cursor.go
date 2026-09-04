package product

import (
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const (
	// productCursorKeyCount は、既定（公開済みのみ）の一覧カーソルが保持するキーの個数（公開日時, ID）です。
	productCursorKeyCount = 2
	// allProductCursorKeyCount は、未公開を含む一覧カーソルが保持するキーの個数（識別子, 登録日時, ID）です。
	allProductCursorKeyCount = 3
	// allProductCursorTag は、未公開を含む一覧カーソルの先頭に置く識別子です。
	// 2 つのモードは並び順の軸が異なるため、一方で得たカーソルを他方へ持ち越すと、公開日時と登録日時を
	// 取り違えたまま解釈できてしまいます。キーの個数が変わることで、その持ち越しが必ず ErrInvalidArgument
	// になります。
	allProductCursorTag = "unpublished"
)

// productCursor は、公開商品一覧（keyset ページネーション）の境界キーを表す usecase 層の値です。
// 直前ページ末尾の公開日時と ID を保持し、次ページ取得時の境界として用います。
// ドメイン層は不透明カーソルを持たず、この境界を primitive（公開日時, ID）で受け取ります。
type productCursor struct {
	publishedAt time.Time
	id          uuid.UUID
}

// allProductCursor は、未公開を含む商品一覧（keyset ページネーション）の境界キーを表す usecase 層の値です。
// 未公開の商品は公開日時を持たないため、境界には登録日時を用います。
type allProductCursor struct {
	createdAt time.Time
	id        uuid.UUID
}

// decodeProductCursor は、cursor の不透明キー列を keyset 境界（productCursor）へ解釈します。
// 先頭ページ（カーソル無し）の場合は nil を返します。キーの個数・型が不正な場合は ErrInvalidArgument を返します。
func decodeProductCursor(cursor *paging.Cursor) (*productCursor, error) {
	if !cursor.HasCursor() {
		return nil, nil //nolint:nilnil // 先頭ページは境界なし（nil）を正常値として返す
	}

	keys := cursor.Keys()
	if len(keys) != productCursorKeyCount {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument,
			"invalid cursor: cursor was issued for a different includeUnpublished")
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

// encodeProductCursor は、現在ページ末尾のソートキー（公開日時, ID）から次ページ用の不透明カーソルを生成します。
// このカーソルを使う一覧は公開済みのみを返すため、PublishedAt は常に非 nil です。
func encodeProductCursor(last *product.Product) string {
	return paging.EncodeCursor(ptr.Deref(last.PublishedAt(), time.Time{}).Format(time.RFC3339Nano), last.ID().String())
}

// decodeAllProductCursor は、cursor の不透明キー列を未公開を含む一覧の keyset 境界へ解釈します。
// 先頭ページ（カーソル無し）の場合は nil を返します。
// 既定の一覧で得たカーソルはキーの個数が異なるため、ここで ErrInvalidArgument になります。
func decodeAllProductCursor(cursor *paging.Cursor) (*allProductCursor, error) {
	if !cursor.HasCursor() {
		return nil, nil //nolint:nilnil // 先頭ページは境界なし（nil）を正常値として返す
	}

	keys := cursor.Keys()
	if len(keys) != allProductCursorKeyCount || keys[0] != allProductCursorTag {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument,
			"invalid cursor: cursor was issued for a different includeUnpublished")
	}

	createdAt, err := time.Parse(time.RFC3339Nano, keys[1])
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: created_at is not RFC3339Nano")
	}

	id, err := uuid.Parse(keys[2])
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: id is not a valid UUID")
	}

	return &allProductCursor{createdAt: createdAt, id: id}, nil
}

// encodeAllProductCursor は、現在ページ末尾のソートキー（登録日時, ID）から次ページ用の不透明カーソルを
// 生成します。既定の一覧のカーソルと取り違えられないよう、先頭に識別子を置きます。
func encodeAllProductCursor(last *product.Product) string {
	return paging.EncodeCursor(
		allProductCursorTag, last.CreatedAt().Format(time.RFC3339Nano), last.ID().String(),
	)
}
