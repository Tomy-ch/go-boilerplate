package system

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		c := NewClock()
		require.NotNil(t, c)
	})
}

func TestClockNow(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		now := NewClock().Now()
		require.WithinDuration(t, time.Now(), now, time.Second)
	})
}

func TestNewSleeper(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		s := NewSleeper()
		require.NotNil(t, s)
	})
}

func TestClockSleep(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定時間経過まで待機して_nilを返す", func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			err := NewSleeper().Sleep(context.Background(), 20*time.Millisecond)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, time.Since(start), 20*time.Millisecond)
		})

		t.Run("待機時間が0以下なら即座にnilを返す", func(t *testing.T) {
			t.Parallel()

			err := NewSleeper().Sleep(context.Background(), 0)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("待機中にctxがキャンセルされたらctxのエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := NewSleeper().Sleep(ctx, time.Hour)
			require.ErrorIs(t, err, context.Canceled)
		})

		t.Run("待機時間が0以下でctxが既にキャンセル済みならctxのエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := NewSleeper().Sleep(ctx, 0)
			require.ErrorIs(t, err, context.Canceled)
		})
	})
}
