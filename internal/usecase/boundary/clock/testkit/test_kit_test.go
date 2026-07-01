package testkit

import (
	"context"
	"testing"
	"time"

	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fatalRecorder は、gomock の期待違反（Errorf / Fatalf）を捕捉するフェイクの TestReporter です。
// Fatalf は panic させ、実テスト（*testing.T）を失敗させずに「呼び出し回数超過」を観測できるようにします。
type fatalRecorder struct {
	failed bool
}

func (r *fatalRecorder) Errorf(string, ...any) { r.failed = true }
func (r *fatalRecorder) Fatalf(string, ...any) {
	r.failed = true
	panic("gomock fatal")
}

func TestNewMockClock(t *testing.T) {
	t.Parallel()

	t.Run("AnyTimesなので複数回呼んでも常に指定した時刻を返す", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		clk := NewMockClock(t, now)

		assert.Equal(t, now, clk.Now())
		assert.Equal(t, now, clk.Now())
	})
}

func TestNewNoopSleeper(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("待機せず常に nil を返す", func(t *testing.T) {
			t.Parallel()

			s := NewNoopSleeper(t)

			require.NoError(t, s.Sleep(context.Background(), time.Hour))
		})
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

	t.Run("Times(1)制約により2回目の呼び出しは期待違反として検出される", func(t *testing.T) {
		t.Parallel()

		// NewMockClockOnce が用いる Times(1) と同じ期待を、フェイク TestReporter 上に組み立てる。
		// 本物の *testing.T へ渡すと 2 回目で実テストごと落ちてしまうため、ここで Times(1) と
		// AnyTimes の差（2 回目を許容するか否か）を安全に実証する。
		now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		rec := &fatalRecorder{}
		ctrl := gomock.NewController(rec)
		// rec は *testing.T ではなく自動 Cleanup が走らないため、期待検証を明示的に締める。
		defer ctrl.Finish()
		clk := mock_clock.NewMockClock(ctrl)
		clk.EXPECT().Now().Return(now).Times(1)

		assert.Equal(t, now, clk.Now())
		require.Panics(t, func() { clk.Now() }, "2回目の呼び出しはFatalfでpanicすること")
		assert.True(t, rec.failed, "gomockが期待違反を報告すること")
	})
}
