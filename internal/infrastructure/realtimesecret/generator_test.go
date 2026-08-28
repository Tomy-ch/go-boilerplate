package realtimesecret

import (
	"encoding/base64"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/xerrors"
)

var errRandom = xerrors.New("random unavailable")

func TestNew(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, New())
}

func Test_cryptoGenerator_Generate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("base64url として復号でき 256 bit ぶんの長さを持つ", func(t *testing.T) {
			t.Parallel()

			got, err := New().Generate()
			require.NoError(t, err)

			decoded, err := base64.RawURLEncoding.DecodeString(got)
			require.NoError(t, err)
			assert.Len(t, decoded, secretBytes)
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

		t.Run("乱数源が必要な長さを返さなければエラーを返す", func(t *testing.T) {
			t.Parallel()

			g := &cryptoGenerator{source: iotest.ErrReader(errRandom)}
			_, err := g.Generate()
			require.ErrorIs(t, err, errRandom)
		})
	})
}
