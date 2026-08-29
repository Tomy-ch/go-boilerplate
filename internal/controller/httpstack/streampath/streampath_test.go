package streampath

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("destination 付きの stream path は true を返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, Is("/v1/streams/inquiry-thread-8f3c"))
		})

		t.Run("destination が空でも接頭辞に一致すれば true を返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, Is("/v1/streams/"))
		})

		t.Run("接頭辞に満たない /v1/streams は false を返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, Is("/v1/streams"))
		})

		t.Run("他の業務エンドポイントは false を返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, Is("/v1/users"))
		})

		t.Run("接頭辞が先頭に無ければ false を返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, Is("/api/v1/streams/x"))
		})
	})
}
