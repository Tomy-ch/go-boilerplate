package realtime_test

import (
	"testing"

	realtimebndry "go-boilerplate/internal/usecase/boundary/realtime"

	"github.com/stretchr/testify/assert"
)

func TestStreamID_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ストリーム識別子をそのまま返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "thread-1", realtimebndry.StreamID("thread-1").String())
		})
	})
}

func TestSequence_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("10 進表記で返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "42", realtimebndry.Sequence(42).String())
		})

		t.Run("先頭ゼロを付けない", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "1", realtimebndry.Sequence(1).String())
		})
	})
}
