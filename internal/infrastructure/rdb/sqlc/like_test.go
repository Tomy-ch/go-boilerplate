package sqlc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapLikePatterns(t *testing.T) {
	t.Parallel()

	t.Run("WrapPrefixLikePattern", func(t *testing.T) {
		t.Parallel()
		got := WrapPrefixLikePattern("hoge")
		assert.Equal(t, "hoge%", got)
	})

	t.Run("WrapSuffixLikePattern", func(t *testing.T) {
		t.Parallel()
		got := WrapSuffixLikePattern("hoge")
		assert.Equal(t, "%hoge", got)
	})

	t.Run("WrapContainsLikePattern", func(t *testing.T) {
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
}
