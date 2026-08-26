package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/pkg/backoff"
)

func Test_newCircuit(t *testing.T) {
	t.Parallel()

	bo := backoff.Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成直後は Closed で cooldown が未確定", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(2, bo)

			assert.Equal(t, phaseClosed, c.phaseNow())
			assert.Zero(t, c.cooldown())
		})

		t.Run("引数の threshold と backoff をそのまま判定材料として保持する", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(2, bo)

			assert.Equal(t, 2, c.threshold)
			assert.Equal(t, bo, c.backoff)
		})
	})
}

func Test_circuit_phaseNow(t *testing.T) {
	t.Parallel()

	bo := backoff.Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Closed / Open / Half-open の遷移に追随して現在の状態を返す", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			assert.Equal(t, phaseClosed, c.phaseNow())

			c.onFailure()
			assert.Equal(t, phaseOpen, c.phaseNow())

			c.toHalfOpen()
			assert.Equal(t, phaseHalfOpen, c.phaseNow())
		})
	})
}

func Test_circuit_cooldown(t *testing.T) {
	t.Parallel()

	bo := backoff.Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一度も Open していない間は 0 を返す", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(2, bo)
			c.onFailure() // 閾値未満なので Closed のまま

			assert.Zero(t, c.cooldown())
		})

		t.Run("Open エピソードごとに backoff 系列の値を返す", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			c.onFailure()
			assert.Equal(t, bo.Duration(0), c.cooldown())

			c.toHalfOpen()
			c.onFailure()

			assert.Equal(t, bo.Duration(1), c.cooldown())
		})
	})
}

func Test_circuit_trip(t *testing.T) {
	t.Parallel()

	bo := backoff.Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Open へ遷移し probing を解除して cooldown を確定する", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			c.mu.Lock()
			c.phase = phaseHalfOpen
			c.probing = true
			c.trip()
			c.mu.Unlock()

			assert.Equal(t, phaseOpen, c.phase)
			assert.False(t, c.probing)
			assert.Equal(t, bo.Duration(0), c.curCooldown)
			assert.Equal(t, 1, c.openCount)
		})

		t.Run("Open を重ねるたび openCount が進み cooldown が指数的に伸びる", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			c.mu.Lock()
			c.trip()
			c.trip()
			c.mu.Unlock()

			assert.Equal(t, bo.Duration(1), c.curCooldown)
			assert.Equal(t, 2, c.openCount)
		})
	})
}
