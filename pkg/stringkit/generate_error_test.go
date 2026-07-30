package stringkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateInRange(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("範囲内の場合は ok=true・空メッセージ", func(t *testing.T) {
			t.Parallel()
			ok, msg := ValidateInRange("こんにちは", 3, 5)
			assert.True(t, ok)
			assert.Empty(t, msg)
		})

		t.Run("範囲外の場合は ok=false・理由メッセージ", func(t *testing.T) {
			t.Parallel()
			ok, msg := ValidateInRange("こんにちは", 1, 3)
			assert.False(t, ok)
			assert.Equal(t, "length must be between 1 and 3 characters (got 5)", msg)
		})

		t.Run("最小文字数未満の場合は ok=false・理由メッセージ", func(t *testing.T) {
			t.Parallel()
			ok, msg := ValidateInRange("あ", 3, 5)
			assert.False(t, ok)
			assert.Equal(t, "length must be between 3 and 5 characters (got 1)", msg)
		})
	})
}

func TestErrorMsgInRange(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("文字列の長さが範囲外の場合、適切なエラーメッセージを返す", func(t *testing.T) {
			t.Parallel()
			lowerBound := 3
			upperBound := 5
			input := "こんにちは"
			expected := "length must be between 3 and 5 characters (got 5)"

			actual := ErrorMsgInRange(lowerBound, upperBound, input)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestErrorMsgMaxOrLess(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("文字列の長さが上限を超えた場合、適切なエラーメッセージを返す", func(t *testing.T) {
			t.Parallel()
			upperBound := 4
			input := "こんにちは"
			expected := "length must be less than or equal to 4 characters (got 5)"

			actual := ErrorMsgMaxOrLess(upperBound, input)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestErrorMsgMinOrMore(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("文字列の長さが下限を下回った場合、適切なエラーメッセージを返す", func(t *testing.T) {
			t.Parallel()
			lowerBound := 6
			input := "こんにちは"
			expected := "length must be greater than or equal to 6 characters (got 5)"

			actual := ErrorMsgMinOrMore(lowerBound, input)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestErrorMsgStrictInRange(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("文字列の長さが厳密な範囲外の場合、適切なエラーメッセージを返す", func(t *testing.T) {
			t.Parallel()
			lowerBound := 3
			upperBound := 5
			input := "こんにちは"
			expected := "length must be greater than 3 and less than 5 characters (got 5)"

			actual := ErrorMsgStrictInRange(lowerBound, upperBound, input)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestErrorMsgLessThanMax(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("文字列の長さが上限以上の場合、適切なエラーメッセージを返す", func(t *testing.T) {
			t.Parallel()
			upperBound := 4
			input := "こんにちは"
			expected := "length must be less than 4 characters (got 5)"

			actual := ErrorMsgLessThanMax(upperBound, input)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestErrorMsgGreaterThanMin(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("文字列の長さが下限以下の場合、適切なエラーメッセージを返す", func(t *testing.T) {
			t.Parallel()
			lowerBound := 6
			input := "こんにちは"
			expected := "length must be greater than 6 characters (got 5)"

			actual := ErrorMsgGreaterThanMin(lowerBound, input)
			assert.Equal(t, expected, actual)
		})
	})
}
