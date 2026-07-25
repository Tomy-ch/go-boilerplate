package product

import (
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/domain/kernel/money"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustPrice は、テスト用に十進文字列から非負の money.Price を構築します。
func mustPrice(t *testing.T, s string) money.Price {
	t.Helper()
	p, err := money.NewPrice(decimaltestkit.MustParse(t, s))
	require.NoError(t, err)
	return p
}

// mustStatusRef は、テスト用に有効な商品ステータス参照を構築します。
func mustStatusRef(t *testing.T, salt, name string) StatusRef {
	t.Helper()
	ref, err := NewStatusRef(uuid.NewTestFromSalt(t, salt), name)
	require.NoError(t, err)
	return ref
}

// mustCategoryRef は、テスト用に有効な商品カテゴリ参照を構築します。
func mustCategoryRef(t *testing.T, salt, name string) CategoryRef {
	t.Helper()
	ref, err := NewCategoryRef(uuid.NewTestFromSalt(t, salt), name)
	require.NoError(t, err)
	return ref
}

func validProductArgs(t *testing.T) (uuid.UUID, string, *string, money.Price, int, *int, StatusRef, CategoryRef, *time.Time, *string) {
	t.Helper()
	id := uuid.NewTestFromSalt(t, "product_id")
	status := mustStatusRef(t, "product_status_id", "在庫あり")
	category := mustCategoryRef(t, "product_category_id", "電子機器")
	publishedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	imagePath := "products/earphone.png"
	return id, "ワイヤレスイヤホン", ptr.To("ノイズキャンセリング対応"), mustPrice(t, "19.99"), 100, ptr.To(10), status, category, ptr.To(publishedAt), ptr.To(imagePath)
}

func TestNew(t *testing.T) {
	t.Parallel()

	id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath := validProductArgs(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全フィールドが有効な場合、Productエンティティが生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath)
			require.NoError(t, err)
			assert.Equal(t, id, actual.ID())
			assert.Equal(t, name, actual.Name())
			assert.Equal(t, description, actual.Description())
			assert.Equal(t, price, actual.Price())
			assert.Equal(t, quantity, actual.Quantity())
			assert.Equal(t, threshold, actual.StockWarningThreshold())
			assert.Equal(t, status, actual.Status())
			assert.Equal(t, category, actual.Category())
			assert.Equal(t, publishedAt, actual.PublishedAt())
			assert.Equal(t, imagePath, actual.ImagePath())

			// ポインタ getter は防御的コピーを返し、返り値を書き換えてもエンティティ内部は不変。
			mutatedPublishedAt := actual.PublishedAt()
			*mutatedPublishedAt = time.Time{}
			require.NotNil(t, actual.PublishedAt())
			assert.Equal(t, *publishedAt, *actual.PublishedAt())

			mutatedImagePath := actual.ImagePath()
			*mutatedImagePath = "mutated"
			require.NotNil(t, actual.ImagePath())
			assert.Equal(t, *imagePath, *actual.ImagePath())
		})

		t.Run("description・stockWarningThreshold・publishedAt・imagePathがnilでも生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, name, nil, price, quantity, nil, status, category, nil, nil)
			require.NoError(t, err)
			assert.Nil(t, actual.Description())
			assert.Nil(t, actual.StockWarningThreshold())
			assert.Nil(t, actual.PublishedAt())
			assert.Nil(t, actual.ImagePath())
		})

		t.Run("priceとquantityが0でも生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, name, description, mustPrice(t, "0"), minQuantity, ptr.To(minThreshold), status, category, publishedAt, imagePath)
			require.NoError(t, err)
			assert.Equal(t, "0", actual.Price().String())
			assert.Equal(t, minQuantity, actual.Quantity())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDがゼロ値の場合、ErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(uuid.UUID{}, name, description, price, quantity, threshold, status, category, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("nameが空の場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, "", description, price, quantity, threshold, status, category, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidName)
		})

		t.Run("nameが最大長を超える場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(
				id,
				strings.Repeat("あ", maxNameLength+1),
				description,
				price,
				quantity,
				threshold,
				status,
				category,
				publishedAt,
				imagePath,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidName)
		})

		t.Run("quantityが負数の場合、ErrInvalidQuantityを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, minQuantity-1, threshold, status, category, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidQuantity)
		})

		t.Run("stockWarningThresholdが負数の場合、ErrInvalidStockWarningThresholdを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, quantity, ptr.To(minThreshold-1), status, category, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStockWarningThreshold)
		})

		t.Run("statusがゼロ値の場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, quantity, threshold, StatusRef{}, category, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("categoryがゼロ値の場合、ErrInvalidCategoryIDを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, quantity, threshold, status, CategoryRef{}, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidCategoryID)
		})
	})
}

func TestProduct_Price(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サブセント精度を保持した価格スナップショットを返す", func(t *testing.T) {
			t.Parallel()

			id, name, description, _, quantity, threshold, status, category, publishedAt, imagePath := validProductArgs(t)
			p, err := New(id, name, description, mustPrice(t, "19.995"), quantity, threshold, status, category, publishedAt, imagePath)
			require.NoError(t, err)
			assert.Equal(t, "19.995", p.Price().String())
			assert.True(t, p.Price().Decimal().Equal(decimaltestkit.MustParse(t, "19.995")))
		})
	})
}
