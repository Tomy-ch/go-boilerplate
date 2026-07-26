package decimal

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_checkMagnitude(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("通常の金額を受理する", func(t *testing.T) {
			t.Parallel()
			d, err := decimal.NewFromString("19.995")
			require.NoError(t, err)
			assert.NoError(t, checkMagnitude(d))
		})

		t.Run("整数桁が上限ちょうどの値を受理する", func(t *testing.T) {
			t.Parallel()
			d := decimal.New(1, maxIntegerDigits-1)
			assert.NoError(t, checkMagnitude(d))
		})

		t.Run("小数桁が上限ちょうどの値を受理する", func(t *testing.T) {
			t.Parallel()
			d := decimal.New(1, -maxFractionDigits)
			assert.NoError(t, checkMagnitude(d))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("整数桁が上限を超える値を拒否する", func(t *testing.T) {
			t.Parallel()
			d := decimal.New(1, maxIntegerDigits)
			require.ErrorIs(t, checkMagnitude(d), ErrInvalid)
		})

		t.Run("小数桁が上限を超える値を拒否する", func(t *testing.T) {
			t.Parallel()
			d := decimal.New(1, -(maxFractionDigits + 1))
			require.ErrorIs(t, checkMagnitude(d), ErrInvalid)
		})

		t.Run("String がプロセスを停止させる規模の指数を拒否する", func(t *testing.T) {
			t.Parallel()
			d, err := decimal.NewFromString("00E100044400")
			require.NoError(t, err)
			require.ErrorIs(t, checkMagnitude(d), ErrInvalid)
		})
	})
}
