package token

import (
	"bytes"
	"encoding/base64"
	"io"
	"testing"
	"testing/iotest"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Generatorを生成する", func(t *testing.T) {
			t.Parallel()
			assert.NotNil(t, New())
		})
	})
}

func Test_cryptoGenerator_Generate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("base64urlとして復号でき乱数バイト数ぶんの長さを持つ", func(t *testing.T) {
			t.Parallel()

			got, err := New().Generate()

			require.NoError(t, err)
			decoded, derr := base64.RawURLEncoding.DecodeString(got)
			require.NoError(t, derr)
			assert.Len(t, decoded, tokenBytes)
		})

		t.Run("ドメインが受け入れる長さの文字列を返す", func(t *testing.T) {
			t.Parallel()
			// 43 文字は 256 ビットを base64url（パディング無し）で表した長さ。
			// ドメインの SessionToken はこの長さを要求するため、両者が食い違うと生成した値が弾かれる。
			got, err := New().Generate()

			require.NoError(t, err)
			assert.Len(t, got, 43)
		})

		t.Run("パディング文字を含まない", func(t *testing.T) {
			t.Parallel()

			got, err := New().Generate()

			require.NoError(t, err)
			assert.NotContains(t, got, "=")
		})

		t.Run("呼ぶたびに異なる値を返す", func(t *testing.T) {
			t.Parallel()
			g := New()
			seen := make(map[string]struct{}, 100)

			for range 100 {
				got, err := g.Generate()
				require.NoError(t, err)
				_, dup := seen[got]
				require.False(t, dup, "生成された値が重複した: %s", got)
				seen[got] = struct{}{}
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("乱数源が読めなければエラーを返す", func(t *testing.T) {
			t.Parallel()
			g := &cryptoGenerator{source: iotest.ErrReader(xerrors.New("entropy source unavailable"))}

			got, err := g.Generate()

			require.Error(t, err)
			assert.Empty(t, got)
		})

		t.Run("乱数源が必要な長さに満たなければエラーを返す", func(t *testing.T) {
			t.Parallel()
			// 短い読み出しを黙って受け入れると、推測できないという保証だけが静かに失われる。
			g := &cryptoGenerator{source: bytes.NewReader(make([]byte, tokenBytes-1))}

			got, err := g.Generate()

			require.ErrorIs(t, err, io.ErrUnexpectedEOF)
			assert.Empty(t, got)
		})
	})
}
