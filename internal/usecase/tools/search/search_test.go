package search

import (
	"strings"
	"testing"

	"go-boilerplate/pkg/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSearchTokens(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("分割・正規化・重複排除・上限適用", func(t *testing.T) {
			t.Parallel()
			input := ptr.To("foo bar_baz  qux foo")
			actual := ParseSearchTokens(input, 10)
			expected := []string{"foo", "bar", "baz", "qux"}
			assert.Equal(t, expected, actual)
		})

		t.Run("maxTokensで切り詰められる", func(t *testing.T) {
			t.Parallel()
			input := ptr.To("a b c d e")
			actual := ParseSearchTokens(input, 3)
			expected := []string{"a", "b", "c"}
			assert.Equal(t, expected, actual)
		})

		t.Run("maxTokensが0以下ならデフォルトが使われる", func(t *testing.T) {
			t.Parallel()
			parts := make([]string, DefaultMaxTokens+2)
			for i := 0; i < len(parts); i++ {
				parts[i] = "t" + strings.Repeat("x", i)
			}
			input := ptr.To(strings.Join(parts, " "))
			actual := ParseSearchTokens(input, 0)
			assert.Len(t, actual, DefaultMaxTokens)
			assert.Equal(t, parts[:DefaultMaxTokens], actual)
		})

		t.Run("空文字列は空配列を返す", func(t *testing.T) {
			t.Parallel()
			actual := ParseSearchTokens(ptr.To(""), 10)
			require.Empty(t, actual)
		})

		t.Run("nilの場合は空配列を返す", func(t *testing.T) {
			t.Parallel()
			actual := ParseSearchTokens(nil, 10)
			require.Empty(t, actual)
		})
	})

	t.Run("境界ケース", func(t *testing.T) {
		t.Parallel()

		t.Run("連続する区切り文字は無視される", func(t *testing.T) {
			t.Parallel()
			input := ptr.To("foo__  _  bar")
			actual := ParseSearchTokens(input, 10)
			expected := []string{"foo", "bar"}
			assert.Equal(t, expected, actual)
		})

		t.Run("MaxKeywordLengthを超える場合は切り詰められる", func(t *testing.T) {
			t.Parallel()

			// MaxKeywordLengthより長い単一トークンを作成
			long := strings.Repeat("a", MaxKeywordLength+10)
			input := ptr.To(long)

			actual := ParseSearchTokens(input, 10)

			// 1トークンで、長さがMaxKeywordLengthに切り詰められていること
			assert.Len(t, actual, 1)
			assert.Len(t, actual[0], MaxKeywordLength)
			assert.Equal(t, strings.Repeat("a", MaxKeywordLength), actual[0])
		})
	})
}

func TestSplitIntoTerms(t *testing.T) {
	t.Parallel()

	t.Run("アンダースコアで分割", func(t *testing.T) {
		t.Parallel()
		actual := splitIntoTerms("a_b_c")
		assert.Equal(t, []string{"a", "b", "c"}, actual)
	})

	t.Run("空白で分割", func(t *testing.T) {
		t.Parallel()
		actual := splitIntoTerms("a b\tc\n d")
		assert.Equal(t, []string{"a", "b", "c", "d"}, actual)
	})

	t.Run("連続する区切り文字は無視される", func(t *testing.T) {
		t.Parallel()
		actual := splitIntoTerms("a__  _b")
		assert.Equal(t, []string{"a", "b"}, actual)
	})
}

func TestTrimAndDropEmpty(t *testing.T) {
	t.Parallel()

	t.Run("前後空白を削除し空要素を排除", func(t *testing.T) {
		t.Parallel()
		in := []string{" a ", "", "  ", "b"}
		actual := trimAndDropEmpty(in)
		assert.Equal(t, []string{"a", "b"}, actual)
	})

	t.Run("空入力は空出力", func(t *testing.T) {
		t.Parallel()
		actual := trimAndDropEmpty([]string{})
		require.Empty(t, actual)
	})
}

func TestDedupePreserveOrder(t *testing.T) {
	t.Parallel()

	t.Run("重複を先頭出現順で除去", func(t *testing.T) {
		t.Parallel()
		in := []string{"a", "b", "a", "c", "b"}
		actual := dedupePreserveOrder(in)
		assert.Equal(t, []string{"a", "b", "c"}, actual)
	})

	t.Run("重複なしはそのまま返る", func(t *testing.T) {
		t.Parallel()
		in := []string{"x", "y", "z"}
		actual := dedupePreserveOrder(in)
		assert.Equal(t, in, actual)
	})
}

func TestLimit(t *testing.T) {
	t.Parallel()

	t.Run("要素数が上限以下", func(t *testing.T) {
		t.Parallel()
		in := []string{"a", "b"}
		actual := limit(in, 5)
		assert.Equal(t, in, actual)
	})

	t.Run("要素数が上限を超える", func(t *testing.T) {
		t.Parallel()
		in := []string{"a", "b", "c", "d"}
		actual := limit(in, 2)
		assert.Equal(t, []string{"a", "b"}, actual)
	})
}
