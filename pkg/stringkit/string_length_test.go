package stringkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuneCount(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("文字列が英字の場合、文字数を正しくカウントする", func(t *testing.T) {
			input := "hello"
			expected := 5

			actual := RuneCount(input)
			assert.Equal(t, expected, actual)
		})

		t.Run("文字列が日本語の場合、文字数を正しくカウントする", func(t *testing.T) {
			input := "こんにちは"
			expected := 5

			actual := RuneCount(input)
			assert.Equal(t, expected, actual)
		})

		t.Run("文字列が空の場合、文字数が0になる", func(t *testing.T) {
			input := ""
			expected := 0

			actual := RuneCount(input)
			assert.Equal(t, expected, actual)
		})

		t.Run("文字列が絵文字の場合、文字数を正しくカウントする", func(t *testing.T) {
			input := "👋🌍"
			expected := 2

			actual := RuneCount(input)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestInRange(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("文字列の長さが範囲内の場合、trueを返す", func(t *testing.T) {
			input := "hello"
			minBound := 1
			maxBound := 5
			expected := true

			actual := InRange(input, minBound, maxBound)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("文字列の長さが範囲外の場合、falseを返す", func(t *testing.T) {
			input := "こんにちは"
			minBound := 1
			maxBound := 4
			expected := false

			actual := InRange(input, minBound, maxBound)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestMaxOrLess(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("文字列の長さが最大値以下の場合、trueを返す", func(t *testing.T) {
			input := "hello"
			maxBound := 5
			expected := true

			actual := MaxOrLess(input, maxBound)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("文字列の長さが最大値を超える場合、falseを返す", func(t *testing.T) {
			input := "こんにちは"
			maxBound := 4
			expected := false

			actual := MaxOrLess(input, maxBound)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestMinOrMore(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("文字列の長さが最小値以上の場合、trueを返す", func(t *testing.T) {
			input := "hello"
			minBound := 5
			expected := true

			actual := MinOrMore(input, minBound)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("文字列の長さが最小値未満の場合、falseを返す", func(t *testing.T) {
			input := "こんにちは"
			minBound := 6
			expected := false

			actual := MinOrMore(input, minBound)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestStrictInRange(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("文字列の長さが範囲内の場合、trueを返す", func(t *testing.T) {
			input := "hello"
			minBound := 1
			maxBound := 6
			expected := true

			actual := StrictInRange(input, minBound, maxBound)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("文字列の長さが範囲外の場合、falseを返す", func(t *testing.T) {
			input := "こんにちは"
			minBound := 1
			maxBound := 5
			expected := false

			actual := StrictInRange(input, minBound, maxBound)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestLessThanMax(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("文字列の長さが最大値未満の場合、trueを返す", func(t *testing.T) {
			input := "hello"
			maxBound := 6
			expected := true

			actual := LessThanMax(input, maxBound)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("文字列の長さが最大値以上の場合、falseを返す", func(t *testing.T) {
			input := "こんにちは"
			maxBound := 5
			expected := false

			actual := LessThanMax(input, maxBound)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestGreaterThanMin(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("文字列の長さが最小値より大きい場合、trueを返す", func(t *testing.T) {
			input := "hello"
			minBound := 4
			expected := true

			actual := GreaterThanMin(input, minBound)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("文字列の長さが最小値以下の場合、falseを返す", func(t *testing.T) {
			input := "こんにちは"
			minBound := 6
			expected := false

			actual := GreaterThanMin(input, minBound)
			assert.Equal(t, expected, actual)
		})
	})
}
