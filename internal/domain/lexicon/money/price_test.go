package money

import (
	"testing"

	"go-boilerplate/pkg/decimal"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrice(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サブセント精度の非負値を受理する", func(t *testing.T) {
			t.Parallel()
			p, err := NewPrice(decimaltestkit.MustParse(t, "19.995"))
			require.NoError(t, err)
			assert.Equal(t, "19.995", p.String())
		})

		t.Run("ゼロを受理する", func(t *testing.T) {
			t.Parallel()
			p, err := NewPrice(decimal.FromInt(0))
			require.NoError(t, err)
			assert.Equal(t, "0", p.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("負値は ErrNegativePrice を返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewPrice(decimaltestkit.MustParse(t, "-0.01"))
			require.ErrorIs(t, err, ErrNegativePrice)
		})

		t.Run("決済スケールへ落とせない大きさは ErrPriceOutOfRange を返す", func(t *testing.T) {
			t.Parallel()
			// 決済スケール（最小単位整数）の幅を 1 だけ超える値。構築を許すと、変換を試みる時点まで
			// 不正が持ち越され、その時点の呼び出し元は拒否する術を持たない。
			_, err := NewPrice(decimaltestkit.MustParse(t, "92233720368547758.08"))
			require.ErrorIs(t, err, ErrPriceOutOfRange)
		})

		t.Run("決済スケールの上限ちょうどは受理する", func(t *testing.T) {
			t.Parallel()
			p, err := NewPrice(decimaltestkit.MustParse(t, "92233720368547758.07"))
			require.NoError(t, err)
			assert.Equal(t, "92233720368547758.07", p.String())
		})
	})
}

func TestPrice_Decimal(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保持している十進量を返す", func(t *testing.T) {
			t.Parallel()
			p, err := NewPrice(decimaltestkit.MustParse(t, "19.99"))
			require.NoError(t, err)
			assert.True(t, p.Decimal().Equal(decimaltestkit.MustParse(t, "19.99")))
		})
	})
}

func TestPrice_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("十進文字列を返す", func(t *testing.T) {
			t.Parallel()
			p, err := NewPrice(decimaltestkit.MustParse(t, "1234.5"))
			require.NoError(t, err)
			assert.Equal(t, "1234.5", p.String())
		})
	})
}

func TestPrice_ToMinorUnit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("セント精度へ丸めた決済スケール整数を返す", func(t *testing.T) {
			t.Parallel()
			p, err := NewPrice(decimaltestkit.MustParse(t, "19.995"))
			require.NoError(t, err)
			minor, err := p.ToMinorUnit(2)
			require.NoError(t, err)
			assert.Equal(t, int64(2000), minor)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時の想定より細かい最小単位では ErrOverflow を返す", func(t *testing.T) {
			t.Parallel()
			// 構築時に保証されるのは maxMinorUnitDigits までの変換であり、それより細かい桁を要求すれば
			// 範囲外になり得る。この経路が残っていることが、構築時の検証が桁数 policy を奪っていない証拠。
			p, err := NewPrice(decimaltestkit.MustParse(t, "92233720368547758.07"))
			require.NoError(t, err)
			_, err = p.ToMinorUnit(maxMinorUnitDigits + 1)
			require.ErrorIs(t, err, decimal.ErrOverflow)
		})

		t.Run("最小単位の桁数が負の場合は ErrInvalidMinorUnit を返す", func(t *testing.T) {
			t.Parallel()
			p, err := NewPrice(decimaltestkit.MustParse(t, "19.99"))
			require.NoError(t, err)
			_, err = p.ToMinorUnit(-1)
			require.ErrorIs(t, err, ErrInvalidMinorUnit)
		})
	})
}
