package httpclient

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testBreakerConfig() BreakerConfig {
	return BreakerConfig{
		FailureThreshold: 0.5,
		MinRequests:      4,
		OpenDuration:     10 * time.Second,
		HalfOpenProbes:   2,
	}
}

func TestBreakerAllowAndRecord(t *testing.T) {
	t.Parallel()

	base := time.Unix(1000, 0)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MinRequests未満では失敗が続いてもclosedのまま", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			for range 3 {
				b.record(false, base)
			}
			assert.Equal(t, breakerClosed, b.currentState())
			assert.True(t, b.allow(base))
		})

		t.Run("MinRequests以上かつ失敗率が閾値以上でopenに倒れfail-fastする", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			for range 4 {
				b.record(false, base)
			}
			assert.Equal(t, breakerOpen, b.currentState())
			assert.False(t, b.allow(base)) // OpenDuration 未経過は通さない
		})

		t.Run("OpenDuration経過でhalf-openへ遷移しプローブを通す", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			for range 4 {
				b.record(false, base)
			}

			assert.True(t, b.allow(base.Add(10*time.Second)))
			assert.Equal(t, breakerHalfOpen, b.currentState())
		})

		t.Run("half-openでHalfOpenProbes件成功するとclosedへ戻る", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			for range 4 {
				b.record(false, base)
			}
			now := base.Add(10 * time.Second)

			// HalfOpenProbes(2) 件のプローブを通して全て成功させる。
			assert.True(t, b.allow(now))
			b.record(true, now)
			assert.True(t, b.allow(now))
			b.record(true, now)

			assert.Equal(t, breakerClosed, b.currentState())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("half-openでプローブが失敗すると再びopenへ倒れる", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			for range 4 {
				b.record(false, base)
			}
			now := base.Add(10 * time.Second)

			assert.True(t, b.allow(now))
			b.record(false, now)

			assert.Equal(t, breakerOpen, b.currentState())
		})

		t.Run("half-openでHalfOpenProbesを超えるプローブは通さない", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			for range 4 {
				b.record(false, base)
			}
			now := base.Add(10 * time.Second)

			assert.True(t, b.allow(now))  // probe 1
			assert.True(t, b.allow(now))  // probe 2
			assert.False(t, b.allow(now)) // 上限超過
		})
	})
}

func TestBreakerManagerGet(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同じDownstreamには同一breakerを返す", func(t *testing.T) {
			t.Parallel()

			m := newBreakerManager()
			b1 := m.get("a", testBreakerConfig())
			b2 := m.get("a", testBreakerConfig())
			assert.Same(t, b1, b2)
		})

		t.Run("異なるDownstreamには別breakerを返す", func(t *testing.T) {
			t.Parallel()

			m := newBreakerManager()
			b1 := m.get("a", testBreakerConfig())
			b2 := m.get("b", testBreakerConfig())
			assert.NotSame(t, b1, b2)
		})
	})
}
