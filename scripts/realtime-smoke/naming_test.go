package main

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/xerrors"
)

var errRandom = xerrors.New("random unavailable")

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errRandom }

func Test_newRunID(t *testing.T) {
	t.Parallel()

	t.Run("16 進 12 桁を返し、呼び出しごとに異なる", func(t *testing.T) {
		t.Parallel()

		a, err := newRunID(rand.Reader)
		require.NoError(t, err)
		b, err := newRunID(rand.Reader)
		require.NoError(t, err)

		assert.Regexp(t, `^[0-9a-f]{12}$`, a)
		assert.NotEqual(t, a, b)
	})

	t.Run("乱数源の失敗はエラーで返す", func(t *testing.T) {
		t.Parallel()

		_, err := newRunID(failingReader{})
		require.ErrorIs(t, err, errRandom)
	})
}

func Test_names_table(t *testing.T) {
	t.Parallel()

	n := names{runID: "0123456789ab"}

	assert.Equal(t, "gobp_smoke_0123456789ab", n.table())
	assert.Equal(t, strings.ToLower(n.table()), n.table(), "DynamoDB の table 名は小文字")
}

func Test_names_topic(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "gobp-smoke-0123456789ab", names{runID: "0123456789ab"}.topic())
}

func Test_names_queue(t *testing.T) {
	t.Parallel()

	n := names{runID: "0123456789ab"}

	assert.Equal(t, "gobp-smoke-0123456789ab-0", n.queue(0))
	assert.Equal(t, "gobp-smoke-0123456789ab-2", n.queue(2))
	assert.NotEqual(t, n.queue(0), n.queue(1))
}

func Test_names_dlq(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "gobp-smoke-0123456789ab-dlq", names{runID: "0123456789ab"}.dlq())
}
