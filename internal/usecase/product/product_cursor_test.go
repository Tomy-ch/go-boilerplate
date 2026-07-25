package product

import (
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_decodeProductCursor(t *testing.T) {
	t.Parallel()

	validID := uuid.NewTestFromSalt(t, "product_cursor_id")
	validTime := time.Date(2026, time.January, 2, 3, 4, 5, 600000000, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な2キーのカーソルはproductCursorへ復号される", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor(validTime.Format(time.RFC3339Nano), validID.String())
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeProductCursor(cursor)
			require.NoError(t, decErr)
			require.NotNil(t, actual)
			assert.Equal(t, validTime, actual.publishedAt)
			assert.Equal(t, validID, actual.id)
		})

		t.Run("先頭ページ(カーソル無し)の場合はnilを返す", func(t *testing.T) {
			t.Parallel()
			cursor, err := paging.NewCursor(nil, nil)
			require.NoError(t, err)

			actual, decErr := decodeProductCursor(cursor)
			require.NoError(t, decErr)
			assert.Nil(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キー数が2でない場合はErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor("only-one-key")
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeProductCursor(cursor)
			require.ErrorIs(t, decErr, apperror.ErrInvalidArgument)
			assert.Nil(t, actual)
		})

		t.Run("published_atがRFC3339Nanoでない場合はErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor("not-a-time", validID.String())
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeProductCursor(cursor)
			require.ErrorIs(t, decErr, apperror.ErrInvalidArgument)
			assert.Nil(t, actual)
		})

		t.Run("idがUUIDでない場合はErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor(validTime.Format(time.RFC3339Nano), "not-a-uuid")
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeProductCursor(cursor)
			require.ErrorIs(t, decErr, apperror.ErrInvalidArgument)
			assert.Nil(t, actual)
		})
	})
}

func Test_encodeProductCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("末尾行のソートキーから生成したカーソルは同じpublished_at/idへ復号できる", func(t *testing.T) {
			t.Parallel()

			id := uuid.NewTestFromSalt(t, "encode_product_cursor_id")
			publishedAt := time.Date(2026, time.March, 4, 5, 6, 7, 800000000, time.UTC)
			status, err := product.NewStatusRef(uuid.NewTestFromSalt(t, "encode_status"), "在庫あり")
			require.NoError(t, err)
			category, err := product.NewCategoryRef(uuid.NewTestFromSalt(t, "encode_category"), "電子機器")
			require.NoError(t, err)
			last, err := product.New(
				id, "商品名", ptr.To("説明"), mustPrice(t, "5.00"), 3, ptr.To(1),
				status, category, ptr.To(publishedAt), nil,
			)
			require.NoError(t, err)

			encoded := encodeProductCursor(last)
			require.NotEmpty(t, encoded)

			cursor, cErr := paging.NewCursor(&encoded, nil)
			require.NoError(t, cErr)

			decoded, decErr := decodeProductCursor(cursor)
			require.NoError(t, decErr)
			require.NotNil(t, decoded)
			assert.Equal(t, publishedAt, decoded.publishedAt)
			assert.Equal(t, id, decoded.id)
		})
	})
}
