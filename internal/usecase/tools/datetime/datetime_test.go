package datetime_test

import (
	"testing"
	"time"

	"go-boilerplate/internal/usecase/tools/datetime"

	"github.com/stretchr/testify/assert"
)

func TestFormatUTC(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("オフセットを持つロケーションの時刻をUTCへ綴り直す", func(t *testing.T) {
			t.Parallel()

			jst := time.FixedZone("JST", 9*60*60)
			got := datetime.FormatUTC(time.Date(2026, time.September, 1, 19, 0, 0, 0, jst))

			assert.Equal(t, "2026-09-01T10:00:00Z", got)
		})

		t.Run("UTCの時刻はそのまま綴る", func(t *testing.T) {
			t.Parallel()

			got := datetime.FormatUTC(time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC))

			assert.Equal(t, "2026-09-01T10:00:00Z", got)
		})

		t.Run("ナノ秒はそのまま残す", func(t *testing.T) {
			t.Parallel()

			got := datetime.FormatUTC(time.Date(2026, time.September, 1, 10, 0, 0, 123456789, time.UTC))

			assert.Equal(t, "2026-09-01T10:00:00.123456789Z", got)
		})

		t.Run("ゼロ値も同じ規則で綴る", func(t *testing.T) {
			t.Parallel()

			got := datetime.FormatUTC(time.Time{})

			assert.Equal(t, "0001-01-01T00:00:00Z", got)
		})
	})
}
