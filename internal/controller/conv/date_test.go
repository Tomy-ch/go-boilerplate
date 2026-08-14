package conv

import (
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatePtr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定された日付を*time.Timeへ変換する", func(t *testing.T) {
			t.Parallel()

			expected := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)
			actual := DatePtr(&openapi_types.Date{Time: expected})

			require.NotNil(t, actual)
			assert.True(t, expected.Equal(*actual))
		})

		t.Run("nilはそのままnilを返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, DatePtr(nil))
		})

		t.Run("変換後の値は引数と独立していて呼び出し側の書き換えの影響を受けない", func(t *testing.T) {
			t.Parallel()

			src := openapi_types.Date{Time: time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)}
			actual := DatePtr(&src)
			src.Time = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

			require.NotNil(t, actual)
			assert.Equal(t, 2026, actual.Year())
		})
	})
}
