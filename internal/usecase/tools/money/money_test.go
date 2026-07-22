package money

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyRateHalfUp(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("USDセントをJPY円へ換算する", func(t *testing.T) {
			t.Parallel()
			// 100.00 USD (10000 cent) × 150.5 = 15050 JPY
			actual := ApplyRateHalfUp(10000, 150.5, 100)
			assert.Equal(t, int64(15050), actual)
		})

		t.Run("端数はhalf-upで切り上げる", func(t *testing.T) {
			t.Parallel()
			// 1 cent × 150.5 / 100 = 1.505 → 2
			actual := ApplyRateHalfUp(1, 150.5, 100)
			assert.Equal(t, int64(2), actual)
		})

		t.Run("ちょうど0.5はhalf-upで切り上げる", func(t *testing.T) {
			t.Parallel()
			// 1 cent × 150.0 / 100 = 1.5 → 2
			actual := ApplyRateHalfUp(1, 150.0, 100)
			assert.Equal(t, int64(2), actual)
		})

		t.Run("0.5未満は切り捨てる", func(t *testing.T) {
			t.Parallel()
			// 1 cent × 149.0 / 100 = 1.49 → 1
			actual := ApplyRateHalfUp(1, 149.0, 100)
			assert.Equal(t, int64(1), actual)
		})

		t.Run("負値は0から遠い方向へ丸める", func(t *testing.T) {
			t.Parallel()
			// -1 cent × 150.0 / 100 = -1.5 → -2（half-away-from-zero）
			actual := ApplyRateHalfUp(-1, 150.0, 100)
			assert.Equal(t, int64(-2), actual)
		})
	})

	t.Run("境界ケース", func(t *testing.T) {
		t.Parallel()

		t.Run("amountMinorが0なら0を返す", func(t *testing.T) {
			t.Parallel()
			actual := ApplyRateHalfUp(0, 150.5, 100)
			assert.Equal(t, int64(0), actual)
		})

		t.Run("scaleが1なら最小単位差なしで換算する", func(t *testing.T) {
			t.Parallel()
			// 100 × 1.5 / 1 = 150
			actual := ApplyRateHalfUp(100, 1.5, 1)
			assert.Equal(t, int64(150), actual)
		})

		t.Run("int64の乗算を超える大額でもオーバーフローせず換算する", func(t *testing.T) {
			t.Parallel()
			// 中間積 amountMinor × rateFixed = 10^13 × 1.505×10^8 ≈ 1.5×10^21 は int64(≈9.2×10^18)を
			// 超えるが、多倍長で計算するため正しい値になる。
			// 10^13 cent (10^11 USD) × 150.5 = 1.505×10^13 JPY
			actual := ApplyRateHalfUp(10_000_000_000_000, 150.5, 100)
			assert.Equal(t, int64(15_050_000_000_000), actual)
		})

		t.Run("float端数を含むレートでも正しく丸める", func(t *testing.T) {
			t.Parallel()
			// 3 cent × 33.335 / 100 = 1.00005 → 1
			actual := ApplyRateHalfUp(3, 33.335, 100)
			assert.Equal(t, int64(1), actual)
		})

		t.Run("scaleが0以下ならゼロ除算でpanicする", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { ApplyRateHalfUp(100, 1.5, 0) })
		})
	})
}
