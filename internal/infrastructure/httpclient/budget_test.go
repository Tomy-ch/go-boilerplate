package httpclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetryBudget(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("初回refillは初期トークンから始まる", func(t *testing.T) {
			t.Parallel()

			b := newRetryBudget()
			b.refill("d", 0.1)

			consumed := 0
			for b.tryConsume("d") {
				consumed++
			}
			assert.Equal(t, int(retryBudgetInitialTokens), consumed)
			assert.False(t, b.tryConsume("d")) // 残量(0.1)はcost(1.0)未満で消費できない
		})

		t.Run("Downstreamごとにトークンは独立しd1の補充はd2の消費に影響しない", func(t *testing.T) {
			t.Parallel()

			b := newRetryBudget()
			b.refill("d1", 1.0) // d1 のみ補充する

			assert.True(t, b.tryConsume("d1"))  // d1 は補充済みで消費できる
			assert.False(t, b.tryConsume("d2")) // d2 は未補充なので消費できない
		})

		t.Run("refillを重ねると上限maxTokensまで補充され超過しない", func(t *testing.T) {
			t.Parallel()

			b := newRetryBudget()
			for range 100 {
				b.refill("d", 0.5) // 初期2 + 0.5*100 だが上限10で頭打ち
			}

			consumed := 0
			for b.tryConsume("d") {
				consumed++
			}
			assert.Equal(t, int(retryBudgetMaxTokens), consumed)
		})

		t.Run("枯渇後にrefillすると再度消費できる", func(t *testing.T) {
			t.Parallel()

			b := newRetryBudget()
			b.refill("d", 0.1)
			for b.tryConsume("d") {
				continue // 枯渇まで消費する
			}

			assert.False(t, b.tryConsume("d"))
			b.refill("d", 1.0) // 1.0 補充すれば再び1回消費できる
			assert.True(t, b.tryConsume("d"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("refill前の未知Downstreamは消費できない", func(t *testing.T) {
			t.Parallel()

			b := newRetryBudget()
			assert.False(t, b.tryConsume("unknown"))
		})
	})
}
