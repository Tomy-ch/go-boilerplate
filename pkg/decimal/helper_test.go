package decimal

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func Test_fromShopspring(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サブセント精度の値を丸めずそのまま保持する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "19.995", fromShopspring(decimal.RequireFromString("19.995")).String())
		})

		t.Run("float経由では表現できない値も欠落なく保持する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "0.1234567890123456789", fromShopspring(decimal.RequireFromString("0.1234567890123456789")).String())
		})

		t.Run("負値の符号を保持する", func(t *testing.T) {
			t.Parallel()
			got := fromShopspring(decimal.RequireFromString("-0.01"))
			assert.Equal(t, "-0.01", got.String())
			assert.True(t, got.IsNegative())
		})

		t.Run("shopspringのゼロ値はゼロのDecimalになる", func(t *testing.T) {
			t.Parallel()
			got := fromShopspring(decimal.Decimal{})
			assert.True(t, got.IsZero())
			assert.Equal(t, Decimal{}, got)
		})
	})
}
