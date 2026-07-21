package product

import (
	"strings"
	"testing"
	"time"

	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validProductArgs(t *testing.T) (uuid.UUID, string, *string, int, int, *int, uuid.UUID, uuid.UUID, time.Time) {
	t.Helper()
	id := uuid.NewTestFromSalt(t, "product_id")
	statusID := uuid.NewTestFromSalt(t, "product_status_id")
	categoryID := uuid.NewTestFromSalt(t, "product_category_id")
	publishedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	return id, "ワイヤレスイヤホン", ptr.To("ノイズキャンセリング対応"), 1999, 100, ptr.To(10), statusID, categoryID, publishedAt
}

func TestNew(t *testing.T) {
	t.Parallel()

	id, name, description, price, quantity, threshold, statusID, categoryID, publishedAt := validProductArgs(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全フィールドが有効な場合、Productエンティティが生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, name, description, price, quantity, threshold, statusID, categoryID, publishedAt)
			require.NoError(t, err)
			assert.Equal(t, id, actual.ID())
			assert.Equal(t, name, actual.Name())
			assert.Equal(t, description, actual.Description())
			assert.Equal(t, price, actual.Price())
			assert.Equal(t, quantity, actual.Quantity())
			assert.Equal(t, threshold, actual.StockWarningThreshold())
			assert.Equal(t, statusID, actual.StatusID())
			assert.Equal(t, categoryID, actual.CategoryID())
			assert.Equal(t, publishedAt, actual.PublishedAt())
		})

		t.Run("descriptionとstockWarningThresholdがnilでも生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, name, nil, price, quantity, nil, statusID, categoryID, publishedAt)
			require.NoError(t, err)
			assert.Nil(t, actual.Description())
			assert.Nil(t, actual.StockWarningThreshold())
		})

		t.Run("priceとquantityが0でも生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, name, description, minPrice, minQuantity, ptr.To(minThreshold), statusID, categoryID, publishedAt)
			require.NoError(t, err)
			assert.Equal(t, minPrice, actual.Price())
			assert.Equal(t, minQuantity, actual.Quantity())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDがゼロ値の場合、ErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(uuid.UUID{}, name, description, price, quantity, threshold, statusID, categoryID, publishedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("nameが空の場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, "", description, price, quantity, threshold, statusID, categoryID, publishedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidName)
		})

		t.Run("nameが最大長を超える場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, strings.Repeat("あ", maxNameLength+1), description, price, quantity, threshold, statusID, categoryID, publishedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidName)
		})

		t.Run("priceが負数の場合、ErrInvalidPriceを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, minPrice-1, quantity, threshold, statusID, categoryID, publishedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidPrice)
		})

		t.Run("quantityが負数の場合、ErrInvalidQuantityを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, minQuantity-1, threshold, statusID, categoryID, publishedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidQuantity)
		})

		t.Run("stockWarningThresholdが負数の場合、ErrInvalidStockWarningThresholdを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, quantity, ptr.To(minThreshold-1), statusID, categoryID, publishedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStockWarningThreshold)
		})

		t.Run("statusIDがゼロ値の場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, quantity, threshold, uuid.UUID{}, categoryID, publishedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("categoryIDがゼロ値の場合、ErrInvalidCategoryIDを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, quantity, threshold, statusID, uuid.UUID{}, publishedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidCategoryID)
		})

		t.Run("publishedAtがゼロ値の場合、ErrInvalidPublishedAtを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, quantity, threshold, statusID, categoryID, time.Time{})
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidPublishedAt)
		})
	})
}
