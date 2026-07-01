package sqlc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapPrefixLikePattern(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("末尾にワイルドカードを付与し前方一致パターンに変換する", func(t *testing.T) {
			t.Parallel()
			got := WrapPrefixLikePattern("hoge")
			assert.Equal(t, "hoge%", got)
		})
	})
}

func TestWrapSuffixLikePattern(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭にワイルドカードを付与し後方一致パターンに変換する", func(t *testing.T) {
			t.Parallel()
			got := WrapSuffixLikePattern("hoge")
			assert.Equal(t, "%hoge", got)
		})
	})
}

func TestWrapContainsLikePattern(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("前後にワイルドカードを付与し部分一致パターンに変換する", func(t *testing.T) {
			t.Parallel()
			got := WrapContainsLikePattern("hoge")
			assert.Equal(t, "%hoge%", got)
		})
	})
}

func TestEscapeForLike(t *testing.T) {
	t.Parallel()

	t.Run("カスタムエスケープ文字でエスケープされる(#)", func(t *testing.T) {
		t.Parallel()
		input := "a#%_b#"
		expected := "a###%#_b##"
		got := EscapeForLike(input, "#")
		assert.Equal(t, expected, got)
	})

	t.Run("デフォルトエスケープ文字(バックスラッシュ)で%と_がエスケープされる", func(t *testing.T) {
		t.Parallel()
		got := EscapeForLike("a%_b", DefaultLikeEscapeChar)
		assert.Equal(t, "a\\%\\_b", got)
	})
}
