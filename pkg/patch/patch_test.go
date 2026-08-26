package patch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnspecified(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Fieldのゼロ値と等価な未指定フィールドを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, Field[string]{}, Unspecified[string]())
		})
	})
}

func TestNull(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定とは区別されるnull指定フィールドを返す", func(t *testing.T) {
			t.Parallel()
			assert.NotEqual(t, Unspecified[string](), Null[string]())
		})

		t.Run("現在値が非nilでも解決結果がnilになるクリア指定を返す", func(t *testing.T) {
			t.Parallel()
			current := "value"
			assert.Nil(t, Null[string]().Resolve(&current))
		})
	})
}

func TestValue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成後に引数の変数を書き換えても保持する値は変わらない", func(t *testing.T) {
			t.Parallel()

			v := "before"
			f := Value(v)
			v = "after"
			require.Equal(t, "after", v)

			actual := f.Resolve(nil)
			require.NotNil(t, actual)
			assert.Equal(t, "before", *actual)
		})
	})
}

func TestField_IsSpecified(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定の場合、falseを返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, Unspecified[string]().IsSpecified())
		})

		t.Run("null指定の場合、trueを返す", func(t *testing.T) {
			t.Parallel()

			assert.True(t, Null[string]().IsSpecified())
		})

		t.Run("値指定の場合、trueを返す", func(t *testing.T) {
			t.Parallel()

			assert.True(t, Value("value").IsSpecified())
		})

		t.Run("ゼロ値は未指定として扱う", func(t *testing.T) {
			t.Parallel()

			var f Field[string]

			assert.False(t, f.IsSpecified())
		})
	})
}

func TestField_Resolve(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定の場合、currentのポインタをそのまま返す", func(t *testing.T) {
			t.Parallel()

			current := "current"
			actual := Unspecified[string]().Resolve(&current)
			require.NotNil(t, actual)
			assert.Same(t, &current, actual)
		})

		t.Run("未指定でcurrentがnilの場合、nilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, Unspecified[string]().Resolve(nil))
		})

		t.Run("ゼロ値のFieldは未指定として扱われcurrentを据え置く", func(t *testing.T) {
			t.Parallel()

			var f Field[int]
			current := 42
			actual := f.Resolve(&current)
			require.NotNil(t, actual)
			assert.Same(t, &current, actual)
		})

		t.Run("null指定の場合、currentが非nilでもnilを返す", func(t *testing.T) {
			t.Parallel()

			current := "current"
			assert.Nil(t, Null[string]().Resolve(&current))
		})

		t.Run("値指定の場合、currentとは別のポインタで指定値を返す", func(t *testing.T) {
			t.Parallel()

			current := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
			updated := time.Date(2026, time.February, 3, 4, 5, 6, 0, time.UTC)

			actual := Value(updated).Resolve(&current)
			require.NotNil(t, actual)
			assert.Equal(t, updated, *actual)
			assert.NotSame(t, &current, actual)
		})

		t.Run("値指定が返すポインタ経由の書き換えは以降の結果に影響しない", func(t *testing.T) {
			t.Parallel()

			f := Value("value")

			first := f.Resolve(nil)
			require.NotNil(t, first)
			*first = "mutated"

			second := f.Resolve(nil)
			require.NotNil(t, second)
			assert.NotSame(t, first, second)
			assert.Equal(t, "value", *second)
		})
	})
}
