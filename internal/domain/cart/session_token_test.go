package cart

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validTokenString は、規定長の URL-safe な文字列です。
const validTokenString = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKL-_012"

func newTestSessionToken(t *testing.T) SessionToken {
	t.Helper()
	token, err := NewSessionToken(validTokenString)
	require.NoError(t, err)
	return token
}

func TestNewSessionToken(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("規定長のURL-safeな文字列を受け入れる", func(t *testing.T) {
			t.Parallel()
			require.Len(t, validTokenString, sessionTokenLength)

			token, err := NewSessionToken(validTokenString)
			require.NoError(t, err)
			assert.Equal(t, validTokenString, token.Value())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字はErrInvalidSessionTokenを返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewSessionToken("")
			require.ErrorIs(t, err, ErrInvalidSessionToken)
		})

		t.Run("規定長より1文字短いとErrInvalidSessionTokenを返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewSessionToken(strings.Repeat("a", sessionTokenLength-1))
			require.ErrorIs(t, err, ErrInvalidSessionToken)
		})

		t.Run("規定長より1文字長いとErrInvalidSessionTokenを返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewSessionToken(strings.Repeat("a", sessionTokenLength+1))
			require.ErrorIs(t, err, ErrInvalidSessionToken)
		})

		t.Run("base64urlに含まれない記号はErrInvalidSessionTokenを返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewSessionToken(strings.Repeat("a", sessionTokenLength-1) + "+")
			require.ErrorIs(t, err, ErrInvalidSessionToken)
		})

		t.Run("base64のパディング文字はErrInvalidSessionTokenを返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewSessionToken(strings.Repeat("a", sessionTokenLength-1) + "=")
			require.ErrorIs(t, err, ErrInvalidSessionToken)
		})

		t.Run("空白を含むとErrInvalidSessionTokenを返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewSessionToken(strings.Repeat("a", sessionTokenLength-1) + " ")
			require.ErrorIs(t, err, ErrInvalidSessionToken)
		})

		t.Run("マルチバイト文字は長さが規定に一致してもErrInvalidSessionTokenを返す", func(t *testing.T) {
			t.Parallel()
			// 「あ」は UTF-8 で 3 バイトのため、41 バイト + 3 バイトで規定長に一致する。
			_, err := NewSessionToken(strings.Repeat("a", sessionTokenLength-3) + "あ")
			require.ErrorIs(t, err, ErrInvalidSessionToken)
		})
	})
}

func Test_isURLSafe(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("base64urlのアルファベットを受け入れる", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isURLSafe('A'))
			assert.True(t, isURLSafe('Z'))
			assert.True(t, isURLSafe('a'))
			assert.True(t, isURLSafe('z'))
			assert.True(t, isURLSafe('0'))
			assert.True(t, isURLSafe('9'))
			assert.True(t, isURLSafe('-'))
			assert.True(t, isURLSafe('_'))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("標準base64だけの記号を拒否する", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isURLSafe('+'))
			assert.False(t, isURLSafe('/'))
			assert.False(t, isURLSafe('='))
		})

		t.Run("URLで意味を持つ記号を拒否する", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isURLSafe('?'))
			assert.False(t, isURLSafe('&'))
			assert.False(t, isURLSafe('#'))
			assert.False(t, isURLSafe('%'))
		})

		t.Run("境界の隣接文字を拒否する", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isURLSafe('@'))
			assert.False(t, isURLSafe('['))
			assert.False(t, isURLSafe('`'))
			assert.False(t, isURLSafe('{'))
			assert.False(t, isURLSafe('/'))
			assert.False(t, isURLSafe(':'))
		})
	})
}

func TestSessionToken_IsZero(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成を経た値はゼロ値ではない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, newTestSessionToken(t).IsZero())
		})

		t.Run("複合リテラルで組み立てた値はゼロ値と判定する", func(t *testing.T) {
			t.Parallel()
			assert.True(t, SessionToken{}.IsZero())
		})
	})
}

func TestSessionToken_Value(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成に用いた文字列を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, validTokenString, newTestSessionToken(t).Value())
		})

		t.Run("ゼロ値は空文字を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, SessionToken{}.Value())
		})
	})
}
