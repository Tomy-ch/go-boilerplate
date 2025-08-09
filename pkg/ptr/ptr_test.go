package ptr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTo(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int型", func(t *testing.T) {
			t.Parallel()
			v := 42
			actual := To(v)
			require.NotNil(t, actual)
			require.Equal(t, v, *actual)
		})

		t.Run("string型", func(t *testing.T) {
			t.Parallel()
			v := "hello"
			actual := To(v)
			require.NotNil(t, actual)
			require.Equal(t, v, *actual)
		})

		t.Run("bool型", func(t *testing.T) {
			t.Parallel()
			v := true
			actual := To(v)
			require.NotNil(t, actual)
			require.Equal(t, v, *actual)
		})

		t.Run("float64型", func(t *testing.T) {
			t.Parallel()
			v := 3.14
			actual := To(v)
			require.NotNil(t, actual)
			require.InDelta(t, v, *actual, 0)
		})

		t.Run("array型", func(t *testing.T) {
			t.Parallel()
			v := [3]int{1, 2, 3}
			actual := To(v)
			require.NotNil(t, actual)
			require.Equal(t, v, *actual)
		})

		t.Run("struct型", func(t *testing.T) {
			t.Parallel()
			type example struct {
				Field string
			}
			v := example{Field: "test"}
			actual := To(v)
			require.NotNil(t, actual)
			require.Equal(t, v, *actual)
		})
	})
}
