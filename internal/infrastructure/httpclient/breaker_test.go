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

func Test_breaker_allow_record(t *testing.T) {
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

		t.Run("回復後に再びMinRequests件失敗すると再openする", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())

			// 1 サイクル目: open → half-open → 成功で closed へ回復する。
			recordClosedFailures(b, 4, base)
			require.Equal(t, breakerOpen, b.currentState())

			now := base.Add(10 * time.Second)
			allowed, gen := b.allow(now)
			require.True(t, allowed)
			b.record(true, now, gen)
			allowed, gen = b.allow(now)
			require.True(t, allowed)
			b.record(true, now, gen)
			require.Equal(t, breakerClosed, b.currentState())

			// 2 サイクル目: toClosed のリセット後、再び MinRequests 件失敗で再 open する。
			recordClosedFailures(b, 4, now)
			assert.Equal(t, breakerOpen, b.currentState())
		})

		t.Run("open状態でのrecordはno-opで状態がopenのまま不変", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordClosedFailures(b, 4, base)
			require.Equal(t, breakerOpen, b.currentState())

			// open は allow が遷移を司るため record は無視され、状態は変わらない。
			b.record(false, base, b.generation)
			assert.Equal(t, breakerOpen, b.currentState())
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

			// 再 open で openedAt がリセットされるため、OpenDuration 未経過の同時刻 allow は通さない。
			reAllowed, _ := b.allow(now)
			assert.False(t, reAllowed)
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

func Test_breaker_allow(t *testing.T) {
	t.Parallel()

	base := time.Unix(1000, 0)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("closedは常にリクエストを通す", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())

			allowed, gen := b.allow(base)
			assert.True(t, allowed)
			assert.Equal(t, breakerClosed, b.currentState())
			assert.Equal(t, b.generation, gen)
		})

		t.Run("openはOpenDuration経過でhalf-openへ遷移しプローブを通す", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordClosedFailures(b, 4, base) // openへ倒す

			allowed, _ := b.allow(base.Add(10 * time.Second)) // OpenDuration=10s 経過
			assert.True(t, allowed)
			assert.Equal(t, breakerHalfOpen, b.currentState())
		})

		t.Run("half-openはHalfOpenProbes件までプローブを通す", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig()) // HalfOpenProbes=2
			recordClosedFailures(b, 4, base)
			now := base.Add(10 * time.Second)

			allowed1, _ := b.allow(now) // 1件目
			allowed2, _ := b.allow(now) // 2件目
			assert.True(t, allowed1)
			assert.True(t, allowed2)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("openはOpenDuration未経過なら通さずfail-fastする", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordClosedFailures(b, 4, base)

			allowed, _ := b.allow(base) // 未経過
			assert.False(t, allowed)
		})

		t.Run("half-openはHalfOpenProbesを超えるプローブを通さない", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig()) // HalfOpenProbes=2
			recordClosedFailures(b, 4, base)
			now := base.Add(10 * time.Second)

			b.allow(now) // 1件目
			b.allow(now) // 2件目
			allowed3, _ := b.allow(now)
			assert.False(t, allowed3) // 上限超過は通さない
		})
	})
}

func Test_breaker_record(t *testing.T) {
	t.Parallel()

	base := time.Unix(1000, 0)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("closedの成功失敗を集計し閾値到達でopenへ倒す", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig()) // MinRequests=4, FailureThreshold=0.5
			recordClosedFailures(b, 4, base)

			assert.Equal(t, breakerOpen, b.currentState())
		})

		t.Run("half-openで失敗を記録するとopenへ戻る", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordClosedFailures(b, 4, base)
			now := base.Add(10 * time.Second)

			_, gen := b.allow(now)
			b.record(false, now, gen)

			assert.Equal(t, breakerOpen, b.currentState())
		})

		t.Run("half-openでHalfOpenProbes件成功するとclosedへ戻る", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig()) // HalfOpenProbes=2
			recordClosedFailures(b, 4, base)
			now := base.Add(10 * time.Second)

			_, gen1 := b.allow(now)
			b.record(true, now, gen1)
			_, gen2 := b.allow(now)
			b.record(true, now, gen2)

			assert.Equal(t, breakerClosed, b.currentState())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("half-openで別エポックの遅延結果は無視する", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordClosedFailures(b, 4, base)
			now := base.Add(10 * time.Second)

			_, gen := b.allow(now)
			b.record(false, now, gen-1) // 古いエポックの遅延結果

			assert.Equal(t, breakerHalfOpen, b.currentState()) // 無視されopenへ倒れない
		})
	})
}

func Test_breaker_shouldOpen(t *testing.T) {
	t.Parallel()

	base := time.Unix(1000, 0)

	// recordOutcomes は、closed 状態で失敗 fail 件・成功 success 件を記録します。
	recordOutcomes := func(b *breaker, fail, success int) {
		for range fail {
			b.record(false, base, b.generation)
		}
		for range success {
			b.record(true, base, b.generation)
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗率が閾値ちょうど_2失敗2成功_0.5_でopenする", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig()) // MinRequests=4, FailureThreshold=0.5
			recordOutcomes(b, 2, 2)

			assert.Equal(t, breakerOpen, b.currentState())
		})

		t.Run("失敗率が閾値未満_1失敗3成功_0.25_ならclosedのまま", func(t *testing.T) {
			t.Parallel()

			b := newBreaker(testBreakerConfig())
			recordOutcomes(b, 1, 3)

			assert.Equal(t, breakerClosed, b.currentState())
		})
	})
}

func Test_breakerManager_get(t *testing.T) {
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
