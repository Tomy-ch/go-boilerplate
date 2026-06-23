package testkit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMockClock(t *testing.T) {
	t.Parallel()

	t.Run("常に指定した時刻を返す", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		clk := NewMockClock(t, now)

		assert.Equal(t, now, clk.Now())
		assert.Equal(t, now, clk.Now())
	})
}

func TestNewMockClockOnce(t *testing.T) {
	t.Parallel()

	t.Run("1回の呼び出しで指定した時刻を返す", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		clk := NewMockClockOnce(t, now)

		assert.Equal(t, now, clk.Now())
	})
}
