package testkit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stepStart は、StepClock の開始時刻として用いるテスト用の基準時刻です。
var stepStart = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func TestNewStepClock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("開始時刻を初期の現在時刻として保持する", func(t *testing.T) {
			t.Parallel()

			clk := NewStepClock(stepStart, time.Second)

			require.NotNil(t, clk)
			assert.Equal(t, stepStart, clk.Now())
		})
	})
}

func TestStepClock_Now(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Sleep 前は開始時刻を返す", func(t *testing.T) {
			t.Parallel()

			clk := NewStepClock(stepStart, time.Minute)

			assert.Equal(t, stepStart, clk.Now())
		})
	})
}

func TestStepClock_Sleep(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctx が有効なら待機時間に依らず step だけ時刻を進めて nil を返す", func(t *testing.T) {
			t.Parallel()

			step := 3 * time.Second
			clk := NewStepClock(stepStart, step)

			err := clk.Sleep(context.Background(), time.Hour)

			require.NoError(t, err)
			assert.Equal(t, stepStart.Add(step), clk.Now())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctx が既にキャンセル済みなら時刻を進めず error を返す", func(t *testing.T) {
			t.Parallel()

			clk := NewStepClock(stepStart, time.Second)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := clk.Sleep(ctx, time.Second)

			require.ErrorIs(t, err, context.Canceled)
			assert.Equal(t, stepStart, clk.Now())
		})
	})
}
