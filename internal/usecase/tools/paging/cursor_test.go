package paging

import (
	"encoding/base64"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("afterとfirstがnilの場合、デフォルト件数の先頭ページになる", func(t *testing.T) {
			t.Parallel()
			actual, err := NewCursor(nil, nil)

			require.NoError(t, err)
			assert.Equal(t, defaultPerPage, actual.Limit())
			assert.False(t, actual.HasCursor())
			assert.Empty(t, actual.Keys())
		})

		t.Run("afterが空文字の場合、先頭ページとして扱われる", func(t *testing.T) {
			t.Parallel()
			actual, err := NewCursor(ptr.To(""), nil)

			require.NoError(t, err)
			assert.False(t, actual.HasCursor())
		})

		t.Run("firstが0以下の場合、デフォルト件数が使用される", func(t *testing.T) {
			t.Parallel()
			actual, err := NewCursor(nil, ptr.To(0))

			require.NoError(t, err)
			assert.Equal(t, defaultPerPage, actual.Limit())
		})

		t.Run("firstが最大件数を超える場合、最大件数にクランプされる", func(t *testing.T) {
			t.Parallel()
			actual, err := NewCursor(nil, ptr.To(maxPerPage+1))

			require.NoError(t, err)
			assert.Equal(t, maxPerPage, actual.Limit())
		})

		t.Run("有効なカーソルが渡された場合、ソートキーが復号される", func(t *testing.T) {
			t.Parallel()
			cursor := EncodeCursor("2024-01-01T00:00:00Z", "11111111-1111-1111-1111-111111111111")
			actual, err := NewCursor(&cursor, ptr.To(20))

			require.NoError(t, err)
			assert.Equal(t, 20, actual.Limit())
			assert.True(t, actual.HasCursor())
			assert.Equal(t, []string{"2024-01-01T00:00:00Z", "11111111-1111-1111-1111-111111111111"}, actual.Keys())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("afterがbase64として不正な場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			actual, err := NewCursor(ptr.To("!!!not-base64!!!"), nil)

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})

		t.Run("afterがJSON配列として不正な場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			invalid := base64.RawURLEncoding.EncodeToString([]byte("not-json"))
			actual, err := NewCursor(&invalid, nil)

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})

		t.Run("afterが空のキーセットの場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			empty := base64.RawURLEncoding.EncodeToString([]byte("[]"))
			actual, err := NewCursor(&empty, nil)

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})
	})
}

func TestCursor_Getters(t *testing.T) {
	t.Parallel()

	t.Run("Limitが正しい値を返す", func(t *testing.T) {
		t.Parallel()
		c := &Cursor{limit: 30, keys: []string{"k"}}

		assert.Equal(t, 30, c.Limit())
	})

	t.Run("Limit32が正しい値を返す", func(t *testing.T) {
		t.Parallel()
		c := &Cursor{limit: 30, keys: nil}

		assert.Equal(t, int32(30), c.Limit32())
	})

	t.Run("Limit32がmaxPerPageを超える場合はクランプされる", func(t *testing.T) {
		t.Parallel()
		c := &Cursor{limit: maxPerPage + 100, keys: nil}

		assert.Equal(t, int32(maxPerPage), c.Limit32())
	})

	t.Run("HasCursorはキーがある場合にtrueを返す", func(t *testing.T) {
		t.Parallel()
		c := &Cursor{limit: 10, keys: []string{"a"}}

		assert.True(t, c.HasCursor())
	})

	t.Run("HasCursorはキーが無い場合にfalseを返す", func(t *testing.T) {
		t.Parallel()
		c := &Cursor{limit: 10, keys: nil}

		assert.False(t, c.HasCursor())
	})

	t.Run("Keysが復号済みのキーを返す", func(t *testing.T) {
		t.Parallel()
		c := &Cursor{limit: 10, keys: []string{"a", "b"}}

		assert.Equal(t, []string{"a", "b"}, c.Keys())
	})

	t.Run("Keysは先頭ページの場合に空スライスを返す", func(t *testing.T) {
		t.Parallel()
		c := &Cursor{limit: 10, keys: nil}

		assert.Empty(t, c.Keys())
	})

	t.Run("Keysが返すスライスを変更しても内部状態は不変である", func(t *testing.T) {
		t.Parallel()
		c := &Cursor{limit: 10, keys: []string{"a", "b"}}

		got := c.Keys()
		got[0] = "mutated"

		assert.Equal(t, []string{"a", "b"}, c.Keys())
	})
}

func TestEncodeCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数キーをエンコードし、NewCursorで復号できる", func(t *testing.T) {
			t.Parallel()
			encoded := EncodeCursor("a", "b")
			actual, err := NewCursor(&encoded, nil)

			require.NoError(t, err)
			assert.Equal(t, []string{"a", "b"}, actual.Keys())
		})

		t.Run("カンマを含むキーでもエンコード・復号が安全に往復する", func(t *testing.T) {
			t.Parallel()
			encoded := EncodeCursor("a,b", "c")
			actual, err := NewCursor(&encoded, nil)

			require.NoError(t, err)
			assert.Equal(t, []string{"a,b", "c"}, actual.Keys())
		})

		t.Run("キーが無い場合は空文字を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, EncodeCursor())
		})
	})
}
