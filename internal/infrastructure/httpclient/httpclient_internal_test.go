package httpclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Header_clone(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("map と値スライスを深くコピーし元と共有しない", func(t *testing.T) {
			t.Parallel()

			original := Header{"X-Trace": {"a", "b"}}
			cloned := original.clone()

			require.Equal(t, original, cloned)

			// clone を書き換えても original に影響しない。
			cloned["X-Trace"][0] = "mutated"
			cloned["X-New"] = []string{"z"}
			assert.Equal(t, "a", original["X-Trace"][0])
			assert.NotContains(t, original, "X-New")
		})

		t.Run("nilヘッダはnilを返す", func(t *testing.T) {
			t.Parallel()

			var h Header

			assert.Nil(t, h.clone())
		})
	})
}
