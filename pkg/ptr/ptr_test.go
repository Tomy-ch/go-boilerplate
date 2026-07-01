package ptr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTo(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int型の値からポインタを生成する", func(t *testing.T) {
			t.Parallel()
			v := 42
			actual := To(v)
			require.NotNil(t, actual)
			assert.Equal(t, v, *actual)
		})

		t.Run("string型の値からポインタを生成する", func(t *testing.T) {
			t.Parallel()
			v := "hello"
			actual := To(v)
			require.NotNil(t, actual)
			assert.Equal(t, v, *actual)
		})

		t.Run("bool型の値からポインタを生成する", func(t *testing.T) {
			t.Parallel()
			v := true
			actual := To(v)
			require.NotNil(t, actual)
			assert.Equal(t, v, *actual)
		})

		t.Run("float64型の値からポインタを生成する", func(t *testing.T) {
			t.Parallel()
			v := 3.14
			actual := To(v)
			require.NotNil(t, actual)
			assert.InDelta(t, v, *actual, 0)
		})

		t.Run("array型の値からポインタを生成する", func(t *testing.T) {
			t.Parallel()
			v := [3]int{1, 2, 3}
			actual := To(v)
			require.NotNil(t, actual)
			assert.Equal(t, v, *actual)
		})

		t.Run("struct型の値からポインタを生成する", func(t *testing.T) {
			t.Parallel()
			type example struct {
				Field string
			}
			v := example{Field: "test"}
			actual := To(v)
			require.NotNil(t, actual)
			assert.Equal(t, v, *actual)
		})
	})
}

func TestDeref(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ポインタが非nilの場合、その値を返す", func(t *testing.T) {
			t.Parallel()
			v := "value"
			actual := Deref(&v, "fallback")
			assert.Equal(t, "value", actual)
		})

		t.Run("ポインタがnilの場合、fallbackを返す", func(t *testing.T) {
			t.Parallel()
			var v *string
			actual := Deref(v, "fallback")
			assert.Equal(t, "fallback", actual)
		})

		t.Run("ゼロ値を指すポインタはfallbackではなくゼロ値を返す", func(t *testing.T) {
			t.Parallel()
			v := 0
			actual := Deref(&v, 99)
			assert.Equal(t, 0, actual)
		})
	})
}

func TestCopy(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilポインタの場合、同値だが別ポインタを返す", func(t *testing.T) {
			t.Parallel()

			type example struct {
				Field string
			}

			v := &example{Field: "test"}
			actual := Copy(v)
			require.NotNil(t, actual)
			assert.Equal(t, *v, *actual)

			// ポインタが異なることを確認
			assert.NotSame(t, v, actual)
		})

		t.Run("nilポインタの場合、nilを返す", func(t *testing.T) {
			t.Parallel()

			var v *int = nil
			actual := Copy(v)
			assert.Nil(t, actual)
		})
	})
}
