package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRFC3339InLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効なRFC3339形式とロケーション", func(t *testing.T) {
		input := "2025-12-05T09:30:00Z"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 0, loc)

		got, err := ParseRFC3339InLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05 09:30:00"

		empty, err := ParseRFC3339InLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseRFC3339UTCInLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効なRFC3339形式とロケーション", func(t *testing.T) {
		input := "2025-12-05T09:30:00Z"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 0, loc)

		got, err := ParseRFC3339UTCInLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05 09:30:00Z"

		empty, err := ParseRFC3339UTCInLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseRFC3339NanoInLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効なRFC3339Nano形式とロケーション", func(t *testing.T) {
		input := "2025-12-05T09:30:00.123456789Z"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 123456789, loc)

		got, err := ParseRFC3339NanoInLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("ナノ秒なしはエラーになる", func(t *testing.T) {
		input := "2025-12-05T09:30:00"

		empty, err := ParseRFC3339NanoInLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseISO8601InLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効なISO8601形式とロケーション", func(t *testing.T) {
		// Z0700 → +0900 など可変だが、ここでは Z (UTC)
		input := "2025-12-05T09:30:00Z"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 0, loc)

		got, err := ParseISO8601InLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05 09:30:00"

		empty, err := ParseISO8601InLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseDateTimeInLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効な日付と時刻形式とロケーション", func(t *testing.T) {
		input := "2025-12-05 09:30:00"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 0, loc)

		got, err := ParseDateTimeInLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05T09:30:00"

		empty, err := ParseDateTimeInLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseDateOnlyInLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効な日付形式とロケーション", func(t *testing.T) {
		input := "2025-12-05"
		want := time.Date(2025, time.December, 5, 9, 0, 0, 0, loc)

		got, err := ParseDateOnlyInLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05"

		empty, err := ParseDateOnlyInLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseCustomLayoutInLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効なカスタムレイアウトとロケーション", func(t *testing.T) {
		layout := "2006/01/02 15:04"
		input := "2025/12/05 09:30"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 0, loc)

		got, err := ParseCustomLayoutInLocation(layout, input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05 09:30:00"
		layout := "02-Jan-2006 15:04:05"

		empty, err := ParseCustomLayoutInLocation(layout, input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestInLocation_NilLocation(t *testing.T) {
	t.Parallel()

	t.Run("locがnilの場合はpanicせずエラーを返す", func(t *testing.T) {
		t.Parallel()

		empty, err := ParseDateOnlyInLocation("2025-12-05", nil)
		require.Empty(t, empty)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loc must not be nil")
	})
}
