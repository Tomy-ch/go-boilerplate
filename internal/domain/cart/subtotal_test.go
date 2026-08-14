package cart

import (
	"testing"

	"go-boilerplate/pkg/decimal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPurchasableLine(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した数量と単価を保持する", func(t *testing.T) {
			t.Parallel()

			price := newEvalPrice(t, "12.50")

			actual := NewPurchasableLine(3, price)

			assert.Equal(t, 3, actual.quantity)
			assert.Equal(t, price, actual.price)
		})
	})
}

func TestSubtotal(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数量を掛けた明細を合算する", func(t *testing.T) {
			t.Parallel()

			actual, err := Subtotal([]PurchasableLine{
				NewPurchasableLine(2, newEvalPrice(t, "10.00")),
				NewPurchasableLine(1, newEvalPrice(t, "5.00")),
			})
			require.NoError(t, err)

			assert.Equal(t, int64(2500), actual)
		})

		t.Run("合算してから丸めるため明細ごとの丸め誤差が積み上がらない", func(t *testing.T) {
			t.Parallel()

			actual, err := Subtotal([]PurchasableLine{
				NewPurchasableLine(1, newEvalPrice(t, "0.005")),
				NewPurchasableLine(1, newEvalPrice(t, "0.005")),
			})
			require.NoError(t, err)

			// 明細ごとに丸めると 1 + 1 = 2 セント。合算してから丸めれば 0.01 ドル = 1 セント。
			assert.Equal(t, int64(1), actual)
		})

		t.Run("合算対象が無ければ0を返す", func(t *testing.T) {
			t.Parallel()

			actual, err := Subtotal(nil)
			require.NoError(t, err)

			assert.Zero(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("決済スケールへ落とせない大きさならErrSubtotalOutOfRangeを返す", func(t *testing.T) {
			t.Parallel()

			price := newEvalPrice(t, "92233720368547758.07")

			actual, err := Subtotal([]PurchasableLine{
				NewPurchasableLine(1, price),
				NewPurchasableLine(1, price),
			})

			require.ErrorIs(t, err, ErrSubtotalOutOfRange)
			require.ErrorIs(t, err, decimal.ErrOverflow)
			assert.Zero(t, actual)
		})
	})
}
