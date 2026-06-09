package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRFC3339ToLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効なRFC3339形式とロケーション", func(t *testing.T) {
		input := "2025-12-05T09:30:00Z"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 0, loc)

		got, err := ParseRFC3339ToLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05 09:30:00"

		empty, err := ParseRFC3339ToLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseRFC3339UTCToLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効なRFC3339形式とロケーション", func(t *testing.T) {
		input := "2025-12-05T09:30:00Z"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 0, loc)

		got, err := ParseRFC3339UTCToLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05 09:30:00Z"

		empty, err := ParseRFC3339UTCToLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseRFC3339NanoToLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効なRFC3339Nano形式とロケーション", func(t *testing.T) {
		input := "2025-12-05T09:30:00.123456789Z"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 123456789, loc)

		got, err := ParseRFC3339NanoToLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("ナノ秒なしはエラーになる", func(t *testing.T) {
		input := "2025-12-05T09:30:00"

		empty, err := ParseRFC3339NanoToLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseISO8601ToLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効なISO8601形式とロケーション", func(t *testing.T) {
		// Z0700 → +0900 など可変だが、ここでは Z (UTC)
		input := "2025-12-05T09:30:00Z"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 0, loc)

		got, err := ParseISO8601ToLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05 09:30:00"

		empty, err := ParseISO8601ToLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseDateTimeToLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効な日付と時刻形式とロケーション", func(t *testing.T) {
		input := "2025-12-05 09:30:00"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 0, loc)

		got, err := ParseDateTimeToLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05T09:30:00"

		empty, err := ParseDateTimeToLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseDateOnlyToLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効な日付形式とロケーション", func(t *testing.T) {
		input := "2025-12-05"
		want := time.Date(2025, time.December, 5, 9, 0, 0, 0, loc)

		got, err := ParseDateOnlyToLocation(input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05"

		empty, err := ParseDateOnlyToLocation(input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestParseCustomLayoutToLocation(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	t.Run("有効なカスタムレイアウトとロケーション", func(t *testing.T) {
		layout := "2006/01/02 15:04"
		input := "2025/12/05 09:30"
		want := time.Date(2025, time.December, 5, 18, 30, 0, 0, loc)

		got, err := ParseCustomLayoutToLocation(layout, input, loc)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("不正なフォーマット", func(t *testing.T) {
		input := "2025/12/05 09:30:00"
		layout := "02-Jan-2006 15:04:05"

		empty, err := ParseCustomLayoutToLocation(layout, input, loc)
		require.Empty(t, empty)
		require.Error(t, err)
	})
}

func TestToLocation_NilLocation(t *testing.T) {
	t.Parallel()

	t.Run("locがnilの場合はpanicせずエラーを返す", func(t *testing.T) {
		t.Parallel()

		empty, err := ParseDateOnlyToLocation("2025-12-05", nil)
		require.Empty(t, empty)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loc must not be nil")
	})
}
