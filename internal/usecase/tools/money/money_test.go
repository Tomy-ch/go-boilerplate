package money

import (
	"testing"

	"go-boilerplate/pkg/decimal"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyRateHalfUp(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("金額をレートで換算し最小単位0桁へ丸める", func(t *testing.T) {
			t.Parallel()
			// 100 USD × 150.5 = 15050 JPY（JPY は最小単位 1 円 = 0 桁）
			actual, err := ApplyRateHalfUp(decimaltestkit.MustParse(t, "100"), decimaltestkit.MustParse(t, "150.5"), 0)
			require.NoError(t, err)
			assert.Equal(t, int64(15050), actual)
		})

		t.Run("端数はhalf-upで切り上げる", func(t *testing.T) {
			t.Parallel()
			// 0.01 × 150.5 = 1.505 → 2
			actual, err := ApplyRateHalfUp(decimaltestkit.MustParse(t, "0.01"), decimaltestkit.MustParse(t, "150.5"), 0)
			require.NoError(t, err)
			assert.Equal(t, int64(2), actual)
		})

		t.Run("ちょうど0.5はhalf-upで切り上げる", func(t *testing.T) {
			t.Parallel()
			// 0.01 × 150.0 = 1.5 → 2
			actual, err := ApplyRateHalfUp(decimaltestkit.MustParse(t, "0.01"), decimaltestkit.MustParse(t, "150.0"), 0)
			require.NoError(t, err)
			assert.Equal(t, int64(2), actual)
		})

		t.Run("0.5未満は切り捨てる", func(t *testing.T) {
			t.Parallel()
			// 0.01 × 149.0 = 1.49 → 1
			actual, err := ApplyRateHalfUp(decimaltestkit.MustParse(t, "0.01"), decimaltestkit.MustParse(t, "149.0"), 0)
			require.NoError(t, err)
			assert.Equal(t, int64(1), actual)
		})

		t.Run("負値は0から遠い方向へ丸める", func(t *testing.T) {
			t.Parallel()
			// -0.01 × 150.0 = -1.5 → -2（half-away-from-zero）
			actual, err := ApplyRateHalfUp(decimaltestkit.MustParse(t, "-0.01"), decimaltestkit.MustParse(t, "150.0"), 0)
			require.NoError(t, err)
			assert.Equal(t, int64(-2), actual)
		})
	})

	t.Run("境界ケース", func(t *testing.T) {
		t.Parallel()

		t.Run("金額が0なら0を返す", func(t *testing.T) {
			t.Parallel()
			actual, err := ApplyRateHalfUp(decimal.FromInt(0), decimaltestkit.MustParse(t, "150.5"), 0)
			require.NoError(t, err)
			assert.Equal(t, int64(0), actual)
		})

		t.Run("最小単位2桁ならセント精度の整数を返す", func(t *testing.T) {
			t.Parallel()
			// 100 × 1.5 = 150、×10^2 = 15000
			actual, err := ApplyRateHalfUp(decimaltestkit.MustParse(t, "100"), decimaltestkit.MustParse(t, "1.5"), 2)
			require.NoError(t, err)
			assert.Equal(t, int64(15000), actual)
		})

		t.Run("サブセントのレートでも正確に丸める", func(t *testing.T) {
			t.Parallel()
			// 0.03 × 33.335 = 1.00005 → 1
			actual, err := ApplyRateHalfUp(decimaltestkit.MustParse(t, "0.03"), decimaltestkit.MustParse(t, "33.335"), 0)
			require.NoError(t, err)
			assert.Equal(t, int64(1), actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最小単位整数が int64 を超える場合はエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := ApplyRateHalfUp(decimaltestkit.MustParse(t, "1e18"), decimaltestkit.MustParse(t, "100"), 0)
			require.ErrorIs(t, err, decimal.ErrOverflow)
		})
	})
}
