package sqlc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapLikePatterns(t *testing.T) {
	t.Parallel()

	t.Run("前方一致のLIKEパターンに変換する", func(t *testing.T) {
		t.Parallel()
		got := WrapPrefixLikePattern("hoge")
		assert.Equal(t, "hoge%", got)
	})

	t.Run("後方一致のLIKEパターンに変換する", func(t *testing.T) {
		t.Parallel()
		got := WrapSuffixLikePattern("hoge")
		assert.Equal(t, "%hoge", got)
	})

	t.Run("部分一致のLIKEパターンに変換する", func(t *testing.T) {
		t.Parallel()
		got := WrapContainsLikePattern("hoge")
		assert.Equal(t, "%hoge%", got)
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
