package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRFC3339(t *testing.T) {
	t.Parallel()
	t.Run("有効なRFC3339形式", func(t *testing.T) {
		t.Parallel()
		input := "2025-12-05T09:30:00Z"
		want := time.Date(2025, time.December, 5, 9, 30, 0, 0, time.UTC)

		got, err := ParseRFC3339(input)
		require.NoError(t, err)

		assert.True(t, got.Equal(want))
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		t.Parallel()
		input := "2025/12/05 09:30:00"

		empty, err := ParseRFC3339(input)
		require.Error(t, err)
		require.Empty(t, empty)
	})
}

func TestParseRFC3339UTC(t *testing.T) {
	t.Parallel()
	t.Run("有効なRFC3339形式（タイムゾーンなし）", func(t *testing.T) {
		t.Parallel()
		input := "2025-12-05T09:30:00Z"
		want := time.Date(2025, time.December, 5, 9, 30, 0, 0, time.UTC)

		got, err := ParseRFC3339UTC(input)
		require.NoError(t, err)

		assert.True(t, got.Equal(want))
	})
	t.Run("不正なフォーマット", func(t *testing.T) {
		t.Parallel()
		input := "2025/12/05 09:30:00Z"

		empty, err := ParseRFC3339UTC(input)
		require.Error(t, err)
		require.Empty(t, empty)
	})
}

func TestParseRFC3339Nano(t *testing.T) {
	t.Parallel()
	t.Run("有効なRFC3339Nano形式", func(t *testing.T) {
		t.Parallel()
		input := "2025-12-05T09:30:00.123456789Z"
		want := time.Date(2025, time.December, 5, 9, 30, 0, 123456789, time.UTC)

		got, err := ParseRFC3339Nano(input)
		require.NoError(t, err)

		assert.True(t, got.Equal(want))
	})

	t.Run("ナノ秒なしはエラーになる", func(t *testing.T) {
		t.Parallel()
		input := "2025-12-05T09:30:00"

		empty, err := ParseRFC3339Nano(input)
		require.Error(t, err)
		require.Empty(t, empty)
	})
}

func TestParseISO8601(t *testing.T) {
	t.Parallel()
	t.Run("有効なISO8601形式", func(t *testing.T) {
		t.Parallel()
		// Z0700 → +0900 など可変だが、ここでは Z (UTC)
		input := "2025-12-05T09:30:00Z"
		want := time.Date(2025, time.December, 5, 9, 30, 0, 0, time.UTC)

		got, err := ParseISO8601(input)
		require.NoError(t, err)

		assert.True(t, got.Equal(want))
	})

	t.Run("自然文形式はエラーになる", func(t *testing.T) {
		t.Parallel()
		input := "Fri Dec 05 09:30:00 2025"

		empty, err := ParseISO8601(input)
		require.Error(t, err)
		require.Empty(t, empty)
	})
}

func TestParseDateTime(t *testing.T) {
	t.Parallel()
	t.Run("日付と時刻のフォーマットが有効", func(t *testing.T) {
		t.Parallel()
		input := "2025-12-05 09:30:00"
		want := time.Date(2025, time.December, 5, 9, 30, 0, 0, time.UTC) // 実装側でUTC/Localを揃える前提

		got, err := ParseDateTime(input)
		require.NoError(t, err)

		assert.True(t, got.Equal(want))
	})

	t.Run("日付のみはエラーになる", func(t *testing.T) {
		t.Parallel()
		input := "2025-12-05"

		empty, err := ParseDateTime(input)
		require.Error(t, err)
		require.Empty(t, empty)
	})
}

func TestParseDateOnly(t *testing.T) {
	t.Parallel()
	t.Run("日付のみのフォーマットが有効", func(t *testing.T) {
		t.Parallel()
		input := "2025-12-05"
		want := time.Date(2025, time.December, 5, 0, 0, 0, 0, time.UTC)

		got, err := ParseDateOnly(input)
		require.NoError(t, err)

		assert.True(t, got.Equal(want))
	})

	t.Run("時刻付きはエラーになる", func(t *testing.T) {
		t.Parallel()
		input := "2025-12-05 09:30:00"

		empty, err := ParseDateOnly(input)
		require.Error(t, err)
		require.Empty(t, empty)
	})
}

func TestParseCustomLayout(t *testing.T) {
	t.Parallel()
	layout := "2006/01/02 15:04"

	t.Run("カスタムレイアウトで有効な形式", func(t *testing.T) {
		t.Parallel()
		input := "2025/12/05 09:30"
		want := time.Date(2025, time.December, 5, 9, 30, 0, 0, time.UTC)

		got, err := ParseCustomLayout(layout, input)
		require.NoError(t, err)

		assert.True(t, got.Equal(want))
	})

	t.Run("カスタムレイアウトに合わない形式はエラー", func(t *testing.T) {
		t.Parallel()
		input := "2025-12-05"

		empty, err := ParseCustomLayout(layout, input)
		require.Error(t, err)
		require.Empty(t, empty)
	})
}
