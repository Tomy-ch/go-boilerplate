package cart

import (
	"testing"
	"time"

	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSubtotalCart は、指定した明細を持つゲストカートを作ります。
func newSubtotalCart(t *testing.T, items ...CartItem) *Cart {
	t.Helper()
	token := newTestSessionToken(t)
	c, err := Reconstruct(uuidtestkit.NewTestFromSalt(t, "subtotal_cart"), Attributes{
		SessionToken: &token,
		Items:        items,
		ExpiresAt:    time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	return c
}

// newSubtotalItem は、商品 ID を指定した明細を作ります。
func newSubtotalItem(t *testing.T, salt string, productID uuid.UUID, quantity int) CartItem {
	t.Helper()

	return NewCartItem(uuidtestkit.NewTestFromSalt(t, salt), CartItemAttributes{
		ProductID: productID,
		Quantity:  quantity,
		AddedAt:   time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
	})
}

func TestCart_Subtotal(t *testing.T) {
	t.Parallel()

	productA := uuidtestkit.NewTestFromSalt(t, "subtotal_a")
	productB := uuidtestkit.NewTestFromSalt(t, "subtotal_b")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数量を掛けた明細を合算する", func(t *testing.T) {
			t.Parallel()

			c := newSubtotalCart(t,
				newSubtotalItem(t, "sub_a", productA, 2),
				newSubtotalItem(t, "sub_b", productB, 1),
			)

			actual, err := c.Subtotal(map[uuid.UUID]ProductSnapshot{
				productA: *newEvalSnapshot(t, "10.00", 5, true),
				productB: *newEvalSnapshot(t, "5.00", 5, true),
			})
			require.NoError(t, err)

			assert.Equal(t, int64(2500), actual)
		})

		t.Run("合算してから丸めるため明細ごとの丸め誤差が積み上がらない", func(t *testing.T) {
			t.Parallel()

			c := newSubtotalCart(t,
				newSubtotalItem(t, "sub_round_a", productA, 1),
				newSubtotalItem(t, "sub_round_b", productB, 1),
			)

			actual, err := c.Subtotal(map[uuid.UUID]ProductSnapshot{
				productA: *newEvalSnapshot(t, "0.005", 5, true),
				productB: *newEvalSnapshot(t, "0.005", 5, true),
			})
			require.NoError(t, err)

			// 明細ごとに丸めると 1 + 1 = 2 セント。合算してから丸めれば 0.01 ドル = 1 セント。
			assert.Equal(t, int64(1), actual)
		})

		t.Run("issue が立った明細は合算に入らない", func(t *testing.T) {
			t.Parallel()

			c := newSubtotalCart(t,
				newSubtotalItem(t, "sub_ok", productA, 1),
				newSubtotalItem(t, "sub_ng", productB, 3),
			)

			actual, err := c.Subtotal(map[uuid.UUID]ProductSnapshot{
				productA: *newEvalSnapshot(t, "10.00", 5, true),
				// 在庫不足で IssueInsufficientStock が立つため、この明細は入らない。
				productB: *newEvalSnapshot(t, "50.00", 1, true),
			})
			require.NoError(t, err)

			assert.Equal(t, int64(1000), actual)
		})

		t.Run("観測値を引けなかった明細は合算に入らない", func(t *testing.T) {
			t.Parallel()

			c := newSubtotalCart(t,
				newSubtotalItem(t, "sub_found", productA, 1),
				newSubtotalItem(t, "sub_missing", productB, 1),
			)

			actual, err := c.Subtotal(map[uuid.UUID]ProductSnapshot{
				productA: *newEvalSnapshot(t, "7.00", 5, true),
			})
			require.NoError(t, err)

			assert.Equal(t, int64(700), actual)
		})

		t.Run("明細が無ければ0を返す", func(t *testing.T) {
			t.Parallel()

			actual, err := newSubtotalCart(t).Subtotal(nil)
			require.NoError(t, err)

			assert.Zero(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("決済スケールへ落とせない大きさならErrSubtotalOutOfRangeを返す", func(t *testing.T) {
			t.Parallel()

			c := newSubtotalCart(t,
				newSubtotalItem(t, "sub_over_a", productA, 1),
				newSubtotalItem(t, "sub_over_b", productB, 1),
			)

			actual, err := c.Subtotal(map[uuid.UUID]ProductSnapshot{
				productA: *newEvalSnapshot(t, "92233720368547758.07", 5, true),
				productB: *newEvalSnapshot(t, "92233720368547758.07", 5, true),
			})

			require.ErrorIs(t, err, ErrSubtotalOutOfRange)
			require.ErrorIs(t, err, decimal.ErrOverflow)
			assert.Zero(t, actual)
		})
	})
}
