package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRFC3339(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効なRFC3339形式をパースできる", func(t *testing.T) {
			t.Parallel()
			input := "2025-12-05T09:30:00Z"
			want := time.Date(2025, time.December, 5, 9, 30, 0, 0, time.UTC)

			got, err := ParseRFC3339(input)
			require.NoError(t, err)

			assert.True(t, got.Equal(want))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正なフォーマットはエラーを返す", func(t *testing.T) {
			t.Parallel()
			empty, err := ParseRFC3339("2025/12/05 09:30:00")
			require.Error(t, err)
			assert.Empty(t, empty)
		})
	})
}

func TestParseRFC3339UTC(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効なRFC3339形式をUTCとしてパースできる", func(t *testing.T) {
			t.Parallel()
			input := "2025-12-05T09:30:00Z"
			want := time.Date(2025, time.December, 5, 9, 30, 0, 0, time.UTC)

			got, err := ParseRFC3339UTC(input)
			require.NoError(t, err)

			assert.True(t, got.Equal(want))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正なフォーマットはエラーを返す", func(t *testing.T) {
			t.Parallel()
			empty, err := ParseRFC3339UTC("2025/12/05 09:30:00Z")
			require.Error(t, err)
			assert.Empty(t, empty)
		})
	})
}

func TestParseRFC3339Nano(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効なRFC3339Nano形式をパースできる", func(t *testing.T) {
			t.Parallel()
			input := "2025-12-05T09:30:00.123456789Z"
			want := time.Date(2025, time.December, 5, 9, 30, 0, 123456789, time.UTC)

			got, err := ParseRFC3339Nano(input)
			require.NoError(t, err)

			assert.True(t, got.Equal(want))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ナノ秒なしはエラーを返す", func(t *testing.T) {
			t.Parallel()
			empty, err := ParseRFC3339Nano("2025-12-05T09:30:00")
			require.Error(t, err)
			assert.Empty(t, empty)
		})
	})
}

func TestParseISO8601(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効なISO8601形式をパースできる", func(t *testing.T) {
			t.Parallel()
			input := "2025-12-05T09:30:00Z"
			want := time.Date(2025, time.December, 5, 9, 30, 0, 0, time.UTC)

			got, err := ParseISO8601(input)
			require.NoError(t, err)

			assert.True(t, got.Equal(want))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("自然文形式はエラーを返す", func(t *testing.T) {
			t.Parallel()
			empty, err := ParseISO8601("Fri Dec 05 09:30:00 2025")
			require.Error(t, err)
			assert.Empty(t, empty)
		})
	})
}

func TestParseDateTime(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("日付と時刻のフォーマットが有効な場合はパースできる", func(t *testing.T) {
			t.Parallel()
			input := "2025-12-05 09:30:00"
			want := time.Date(2025, time.December, 5, 9, 30, 0, 0, time.UTC)

			got, err := ParseDateTime(input)
			require.NoError(t, err)

			assert.True(t, got.Equal(want))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("日付のみはエラーを返す", func(t *testing.T) {
			t.Parallel()
			empty, err := ParseDateTime("2025-12-05")
			require.Error(t, err)
			assert.Empty(t, empty)
		})
	})
}

func TestParseDateOnly(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("日付のみのフォーマットが有効な場合はパースできる", func(t *testing.T) {
			t.Parallel()
			input := "2025-12-05"
			want := time.Date(2025, time.December, 5, 0, 0, 0, 0, time.UTC)

			got, err := ParseDateOnly(input)
			require.NoError(t, err)

			assert.True(t, got.Equal(want))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("時刻付きはエラーを返す", func(t *testing.T) {
			t.Parallel()
			empty, err := ParseDateOnly("2025-12-05 09:30:00")
			require.Error(t, err)
			assert.Empty(t, empty)
		})
	})
}

func TestParseCustomLayout(t *testing.T) {
	t.Parallel()

	layout := "2006/01/02 15:04"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カスタムレイアウトに沿った形式をパースできる", func(t *testing.T) {
			t.Parallel()
			input := "2025/12/05 09:30"
			want := time.Date(2025, time.December, 5, 9, 30, 0, 0, time.UTC)

			got, err := ParseCustomLayout(layout, input)
			require.NoError(t, err)

			assert.True(t, got.Equal(want))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カスタムレイアウトに合わない形式はエラーを返す", func(t *testing.T) {
			t.Parallel()
			empty, err := ParseCustomLayout(layout, "2025-12-05")
			require.Error(t, err)
			assert.Empty(t, empty)
		})
	})
}
