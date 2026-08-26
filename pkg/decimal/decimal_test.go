package decimal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustParse は decimal 本体テスト用のローカルヘルパーです（testkit は decimal を import するため循環回避で本体側には使えない）。
func mustParse(t *testing.T, s string) Decimal {
	t.Helper()
	d, err := Parse(s)
	require.NoError(t, err)
	return d
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サブセント精度の十進文字列を解析する", func(t *testing.T) {
			t.Parallel()
			d, err := Parse("19.995")
			require.NoError(t, err)
			assert.Equal(t, "19.995", d.String())
		})

		t.Run("負値を解析する", func(t *testing.T) {
			t.Parallel()
			d, err := Parse("-0.01")
			require.NoError(t, err)
			assert.Equal(t, "-0.01", d.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("十進として無効な文字列は ErrInvalid を返す", func(t *testing.T) {
			t.Parallel()
			_, err := Parse("abc")
			require.ErrorIs(t, err, ErrInvalid)
		})
	})
}

func TestFromInt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int64 から Decimal を生成する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "1999", FromInt(1999).String())
		})
	})
}

func TestDecimal_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ゼロ値は 0 を返す", func(t *testing.T) {
			t.Parallel()
			var d Decimal
			assert.Equal(t, "0", d.String())
		})
	})
}

func TestDecimal_Add(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("十進誤差なく加算する", func(t *testing.T) {
			t.Parallel()
			got := mustParse(t, "0.1").Add(mustParse(t, "0.2"))
			assert.Equal(t, "0.3", got.String())
		})
	})
}

func TestDecimal_Sub(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("十進誤差なく減算する", func(t *testing.T) {
			t.Parallel()
			got := mustParse(t, "0.3").Sub(mustParse(t, "0.1"))
			assert.Equal(t, "0.2", got.String())
		})
	})
}

func TestDecimal_Mul(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("レート乗算で桁を保持する", func(t *testing.T) {
			t.Parallel()
			got := mustParse(t, "100").Mul(mustParse(t, "150.5"))
			assert.Equal(t, "15050", got.String())
		})
	})
}

func TestDecimal_Neg(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("符号を反転する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "-1.5", mustParse(t, "1.5").Neg().String())
		})
	})
}

func TestDecimal_DivRound(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定桁で 0 から遠い方向へ丸めた商を返す", func(t *testing.T) {
			t.Parallel()
			got := mustParse(t, "10").DivRound(mustParse(t, "3"), 2)
			assert.Equal(t, "3.33", got.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ゼロ除算は事前条件違反として panic する", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { _ = mustParse(t, "10").DivRound(FromInt(0), 2) })
		})
	})
}

func TestDecimal_RoundHalfAwayFromZero(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正値の中間は 0 から遠い方向へ丸める", func(t *testing.T) {
			t.Parallel()
			got, err := mustParse(t, "19.995").RoundHalfAwayFromZero(2).ToScaledInt64(2)
			require.NoError(t, err)
			assert.Equal(t, int64(2000), got)
		})

		t.Run("負値の中間は 0 から遠い方向へ丸める", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "-2", mustParse(t, "-1.5").RoundHalfAwayFromZero(0).String())
		})
	})
}

func TestDecimal_Truncate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定桁で 0 方向へ切り捨てる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "19.99", mustParse(t, "19.999").Truncate(2).String())
		})
	})
}

func TestDecimal_Cmp(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("大小関係を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, -1, mustParse(t, "1").Cmp(mustParse(t, "2")))
			assert.Equal(t, 0, mustParse(t, "1.0").Cmp(mustParse(t, "1")))
			assert.Equal(t, 1, mustParse(t, "2").Cmp(mustParse(t, "1")))
		})
	})
}

func TestDecimal_Equal(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スケール違いでも数値が等しければ真", func(t *testing.T) {
			t.Parallel()
			assert.True(t, mustParse(t, "1.50").Equal(mustParse(t, "1.5")))
		})
	})
}

func TestDecimal_Sign(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("符号を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 1, mustParse(t, "0.1").Sign())
			assert.Equal(t, 0, FromInt(0).Sign())
			assert.Equal(t, -1, mustParse(t, "-0.1").Sign())
		})
	})
}

func TestDecimal_IsZero(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("0 判定を返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, FromInt(0).IsZero())
			assert.False(t, mustParse(t, "0.001").IsZero())
		})
	})
}

func TestDecimal_IsNegative(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("負値判定を返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, mustParse(t, "-0.01").IsNegative())
			assert.False(t, FromInt(0).IsNegative())
			assert.False(t, mustParse(t, "0.01").IsNegative())
		})
	})
}

func TestDecimal_ToScaledInt64(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("セント精度へ変換する", func(t *testing.T) {
			t.Parallel()
			got, err := mustParse(t, "19.99").ToScaledInt64(2)
			require.NoError(t, err)
			assert.Equal(t, int64(1999), got)
		})

		t.Run("サブセントは 0 から遠い方向へ丸める", func(t *testing.T) {
			t.Parallel()
			got, err := mustParse(t, "19.995").ToScaledInt64(2)
			require.NoError(t, err)
			assert.Equal(t, int64(2000), got)
		})

		t.Run("負のサブセントは 0 から遠い方向へ丸める", func(t *testing.T) {
			t.Parallel()
			got, err := mustParse(t, "-19.995").ToScaledInt64(2)
			require.NoError(t, err)
			assert.Equal(t, int64(-2000), got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int64 範囲外は ErrOverflow を返す", func(t *testing.T) {
			t.Parallel()
			_, err := mustParse(t, "9223372036854775808").ToScaledInt64(0)
			require.ErrorIs(t, err, ErrOverflow)
		})
	})
}

func TestDecimal_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON 文字列として符号化する", func(t *testing.T) {
			t.Parallel()
			b, err := json.Marshal(mustParse(t, "19.99"))
			require.NoError(t, err)
			assert.JSONEq(t, `"19.99"`, string(b))
		})
	})
}

func TestDecimal_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON 文字列から復元する", func(t *testing.T) {
			t.Parallel()
			var d Decimal
			require.NoError(t, json.Unmarshal([]byte(`"150.5"`), &d))
			assert.Equal(t, "150.5", d.String())
		})

		t.Run("JSON number も桁を保持して復元する", func(t *testing.T) {
			t.Parallel()
			var d Decimal
			require.NoError(t, json.Unmarshal([]byte(`150.5`), &d))
			assert.Equal(t, "150.5", d.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("無効な JSON は ErrInvalid を返す", func(t *testing.T) {
			t.Parallel()
			var d Decimal
			require.ErrorIs(t, d.UnmarshalJSON([]byte(`"abc"`)), ErrInvalid)
		})
	})
}

func TestDecimal_Scan(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("文字列の NUMERIC を読み込む", func(t *testing.T) {
			t.Parallel()
			var d Decimal
			require.NoError(t, d.Scan("19.995"))
			assert.Equal(t, "19.995", d.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非対応の型は ErrInvalid を返す", func(t *testing.T) {
			t.Parallel()
			var d Decimal
			require.ErrorIs(t, d.Scan([]struct{}{}), ErrInvalid)
		})
	})
}

func TestDecimal_Value(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Scan と往復できる文字列値を返す", func(t *testing.T) {
			t.Parallel()
			v, err := mustParse(t, "19.995").Value()
			require.NoError(t, err)
			var back Decimal
			require.NoError(t, back.Scan(v))
			assert.Equal(t, "19.995", back.String())
		})
	})
}
