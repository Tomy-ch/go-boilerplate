package product

import (
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_decodeProductCursor(t *testing.T) {
	t.Parallel()

	validID := uuidtestkit.NewTestFromSalt(t, "product_cursor_id")
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

			id := uuidtestkit.NewTestFromSalt(t, "encode_product_cursor_id")
			publishedAt := time.Date(2026, time.March, 4, 5, 6, 7, 800000000, time.UTC)
			status, err := product.NewStatusRef(uuidtestkit.NewTestFromSalt(t, "encode_status"), "在庫あり")
			require.NoError(t, err)
			category, err := product.NewCategoryRef(uuidtestkit.NewTestFromSalt(t, "encode_category"), "電子機器")
			require.NoError(t, err)
			last, err := product.New(id, product.Attributes{
				Name:                  "商品名",
				Description:           ptr.To("説明"),
				Price:                 mustPrice(t, "5.00"),
				Quantity:              3,
				StockWarningThreshold: ptr.To(1),
				Status:                status,
				Category:              category,
				PublishedAt:           ptr.To(publishedAt),
			}, testCreatedAt)
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

// validCursorProductArgs は、カーソル符号化のテスト用に有効な商品 ID と属性一式を構築します。
func validCursorProductArgs(t *testing.T) (uuid.UUID, product.Attributes) {
	t.Helper()

	status, err := product.NewStatusRef(uuidtestkit.NewTestFromSalt(t, "all_cursor_status"), "在庫あり")
	require.NoError(t, err)
	category, err := product.NewCategoryRef(uuidtestkit.NewTestFromSalt(t, "all_cursor_category"), "電子機器")
	require.NoError(t, err)

	return uuidtestkit.NewTestFromSalt(t, "all_cursor_product_id"), product.Attributes{
		Name:                  "未公開の商品",
		Description:           ptr.To("説明"),
		Price:                 mustPrice(t, "5.00"),
		Quantity:              3,
		StockWarningThreshold: ptr.To(1),
		Status:                status,
		Category:              category,
		PublishedAt:           nil,
	}
}

func Test_decodeAllProductCursor(t *testing.T) {
	t.Parallel()

	validID := uuidtestkit.NewTestFromSalt(t, "all_product_cursor_id")
	validTime := time.Date(2025, time.December, 31, 23, 59, 58, 900000000, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な3キーのカーソルはallProductCursorへ復号される", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor(
				allProductCursorTag, validTime.Format(time.RFC3339Nano), validID.String(),
			)
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeAllProductCursor(cursor)
			require.NoError(t, decErr)
			require.NotNil(t, actual)
			assert.Equal(t, validTime, actual.createdAt)
			assert.Equal(t, validID, actual.id)
		})

		t.Run("先頭ページ(カーソル無し)の場合はnilを返す", func(t *testing.T) {
			t.Parallel()
			cursor, err := paging.NewCursor(nil, nil)
			require.NoError(t, err)

			actual, decErr := decodeAllProductCursor(cursor)
			require.NoError(t, decErr)
			assert.Nil(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既定の可視範囲で得た2キーのカーソルはErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			// 2 つのモードは並び順の軸が異なるため、公開日時を登録日時として解釈させてはならない。
			encoded := paging.EncodeCursor(validTime.Format(time.RFC3339Nano), validID.String())
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeAllProductCursor(cursor)
			require.ErrorIs(t, decErr, apperror.ErrInvalidArgument)
			assert.Nil(t, actual)
		})

		t.Run("キー数は3だが識別子が異なる場合はErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor("published", validTime.Format(time.RFC3339Nano), validID.String())
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeAllProductCursor(cursor)
			require.ErrorIs(t, decErr, apperror.ErrInvalidArgument)
			assert.Nil(t, actual)
		})

		t.Run("created_atがRFC3339Nanoでない場合はErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor(allProductCursorTag, "not-a-time", validID.String())
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeAllProductCursor(cursor)
			require.ErrorIs(t, decErr, apperror.ErrInvalidArgument)
			assert.Nil(t, actual)
		})

		t.Run("idがUUIDでない場合はErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor(
				allProductCursorTag, validTime.Format(time.RFC3339Nano), "not-a-uuid",
			)
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeAllProductCursor(cursor)
			require.ErrorIs(t, decErr, apperror.ErrInvalidArgument)
			assert.Nil(t, actual)
		})
	})
}

func Test_encodeAllProductCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("識別子と登録日時とIDの3キーへ符号化される", func(t *testing.T) {
			t.Parallel()
			id, attrs := validCursorProductArgs(t)
			p, err := product.Reconstruct(id, attrs, 1, testCreatedAt)
			require.NoError(t, err)

			encoded := encodeAllProductCursor(p)
			cursor, curErr := paging.NewCursor(&encoded, nil)
			require.NoError(t, curErr)

			assert.Equal(t,
				[]string{allProductCursorTag, testCreatedAt.Format(time.RFC3339Nano), id.String()},
				cursor.Keys())
		})

		t.Run("符号化したカーソルは既定の可視範囲では復号できない", func(t *testing.T) {
			t.Parallel()
			id, attrs := validCursorProductArgs(t)
			p, err := product.Reconstruct(id, attrs, 1, testCreatedAt)
			require.NoError(t, err)

			encoded := encodeAllProductCursor(p)
			cursor, curErr := paging.NewCursor(&encoded, nil)
			require.NoError(t, curErr)

			actual, decErr := decodeProductCursor(cursor)
			require.ErrorIs(t, decErr, apperror.ErrInvalidArgument)
			assert.Nil(t, actual)
		})
	})
}
