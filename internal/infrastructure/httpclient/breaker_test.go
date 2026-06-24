package httpclient

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testBreakerConfig() BreakerConfig {
	return BreakerConfig{
		FailureThreshold: 0.5,
		MinRequests:      4,
		OpenDuration:     10 * time.Second,
		HalfOpenProbes:   2,
	}
}

// recordClosedFailures は、closed 状態で n 件の失敗を記録します。
func recordClosedFailures(b *breaker, n int, now time.Time) {
	for range n {
		b.record(false, now, b.generation)
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
			recordClosedFailures(b, 3, base)

			assert.Equal(t, breakerClosed, b.currentState())
			allowed, _ := b.allow(base)
			assert.True(t, allowed)
		})

		t.Run("MinRequests以上かつ失敗率が閾値以上でopenに倒れfail-fastする", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordClosedFailures(b, 4, base)

			assert.Equal(t, breakerOpen, b.currentState())
			allowed, _ := b.allow(base) // OpenDuration 未経過は通さない
			assert.False(t, allowed)
		})

		t.Run("OpenDuration経過でhalf-openへ遷移しプローブを通す", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordClosedFailures(b, 4, base)

			allowed, _ := b.allow(base.Add(10 * time.Second))
			assert.True(t, allowed)
			assert.Equal(t, breakerHalfOpen, b.currentState())
		})

		t.Run("half-openでHalfOpenProbes件成功するとclosedへ戻る", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordClosedFailures(b, 4, base)
			now := base.Add(10 * time.Second)

			allowed, gen := b.allow(now)
			require.True(t, allowed)
			b.record(true, now, gen)
			allowed, gen = b.allow(now)
			require.True(t, allowed)
			b.record(true, now, gen)

			assert.Equal(t, breakerClosed, b.currentState())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("half-openでプローブが失敗すると再びopenへ倒れる", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordClosedFailures(b, 4, base)
			now := base.Add(10 * time.Second)

			allowed, gen := b.allow(now)
			require.True(t, allowed)
			b.record(false, now, gen)

			assert.Equal(t, breakerOpen, b.currentState())
		})

		t.Run("half-openでHalfOpenProbesを超えるプローブは通さない", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordClosedFailures(b, 4, base)
			now := base.Add(10 * time.Second)

			a1, _ := b.allow(now)
			a2, _ := b.allow(now)
			a3, _ := b.allow(now)
			assert.True(t, a1)  // probe 1
			assert.True(t, a2)  // probe 2
			assert.False(t, a3) // 上限超過
		})

		t.Run("旧エポックの遅延成功は新エポックのクローズ判定に混入しない", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordClosedFailures(b, 4, base)
			now := base.Add(10 * time.Second)

			// エポック1: 2 プローブを承認し、1 つ目を失敗させて再び open へ倒す。
			_, gen1 := b.allow(now)
			_, genStale := b.allow(now) // 同エポックの遅延プローブ枠
			b.record(false, now, gen1)
			require.Equal(t, breakerOpen, b.currentState())

			// エポック2へ遷移し、新たに 1 プローブを承認する。
			later := now.Add(10 * time.Second)
			allowed, gen3 := b.allow(later)
			require.True(t, allowed)
			require.Equal(t, breakerHalfOpen, b.currentState())

			// 旧エポックの遅延成功はエポック不一致で無視され、新エポックの成功にカウントされない。
			b.record(true, later, genStale)
			assert.Equal(t, breakerHalfOpen, b.currentState())

			// 新エポックの成功は 1 件のみのため、まだ closed へは戻らない。
			b.record(true, later, gen3)
			assert.Equal(t, breakerHalfOpen, b.currentState())
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
