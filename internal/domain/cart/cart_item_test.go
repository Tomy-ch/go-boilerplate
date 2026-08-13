package cart

import (
	"testing"
	"time"

	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/pkg/decimal"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baseTime は、テスト全体で用いる決定的な基準時刻です。
var baseTime = time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

func newTestPrice(t *testing.T, amount int64) money.Price {
	t.Helper()
	price, err := money.NewPrice(decimal.FromInt(amount))
	require.NoError(t, err)
	return price
}

func newTestCartItem(t *testing.T, salt string, quantity int) CartItem {
	t.Helper()
	return NewCartItem(uuidtestkit.NewTestFromSalt(t, "item_"+salt), CartItemAttributes{
		ProductID: uuidtestkit.NewTestFromSalt(t, "product_"+salt),
		Quantity:  quantity,
		AddedAt:   baseTime,
	})
}

func TestNewCartItem(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("属性をそのまま保持する", func(t *testing.T) {
			t.Parallel()
			id := uuidtestkit.NewTestFromSalt(t, "item")
			productID := uuidtestkit.NewTestFromSalt(t, "product")
			price := newTestPrice(t, 10)

			item := NewCartItem(id, CartItemAttributes{
				ProductID:     productID,
				Quantity:      3,
				AddedAt:       baseTime,
				LastSeenPrice: &price,
			})

			assert.Equal(t, id, item.ID())
			assert.Equal(t, productID, item.ProductID())
			assert.Equal(t, 3, item.Quantity())
			assert.Equal(t, baseTime, item.AddedAt())
			require.NotNil(t, item.LastSeenPrice())
			assert.Equal(t, price, *item.LastSeenPrice())
		})

		t.Run("提示価格が未設定でも組み立てられる", func(t *testing.T) {
			t.Parallel()
			item := newTestCartItem(t, "no_price", 1)
			assert.Nil(t, item.LastSeenPrice())
		})

		t.Run("集約の不変条件は判定しない", func(t *testing.T) {
			t.Parallel()
			item := NewCartItem(uuidtestkit.NewTestFromSalt(t, "item"), CartItemAttributes{Quantity: 0})
			assert.Equal(t, 0, item.Quantity())
			assert.True(t, item.ProductID().IsNil())
		})

		t.Run("渡された提示価格のポインタを共有しない", func(t *testing.T) {
			t.Parallel()
			price := newTestPrice(t, 10)
			item := NewCartItem(uuidtestkit.NewTestFromSalt(t, "item"), CartItemAttributes{
				LastSeenPrice: &price,
			})

			price = newTestPrice(t, 999)

			require.NotNil(t, item.LastSeenPrice())
			assert.Equal(t, newTestPrice(t, 10), *item.LastSeenPrice())
		})
	})
}

func TestCartItem_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("組み立てに用いた明細IDを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, uuidtestkit.NewTestFromSalt(t, "item_a"), newTestCartItem(t, "a", 1).ID())
		})
	})
}

func TestCartItem_ProductID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("組み立てに用いた商品IDを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, uuidtestkit.NewTestFromSalt(t, "product_a"), newTestCartItem(t, "a", 1).ProductID())
		})
	})
}

func TestCartItem_Quantity(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("組み立てに用いた数量を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 7, newTestCartItem(t, "a", 7).Quantity())
		})
	})
}

func TestCartItem_AddedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("組み立てに用いた追加時刻を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, baseTime, newTestCartItem(t, "a", 1).AddedAt())
		})
	})
}

func TestCartItem_LastSeenPrice(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未提示ならnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, newTestCartItem(t, "a", 1).LastSeenPrice())
		})

		t.Run("戻り値を書き換えても内部状態は変わらない", func(t *testing.T) {
			t.Parallel()
			price := newTestPrice(t, 10)
			item := NewCartItem(uuidtestkit.NewTestFromSalt(t, "item"), CartItemAttributes{
				LastSeenPrice: &price,
			})

			got := item.LastSeenPrice()
			require.NotNil(t, got)
			*got = newTestPrice(t, 999)

			assert.Equal(t, newTestPrice(t, 10), *item.LastSeenPrice())
		})
	})
}
