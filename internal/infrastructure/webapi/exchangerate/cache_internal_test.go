package exchangerate

import (
	"strconv"
	"testing"
	"time"

	"go-boilerplate/internal/usecase/boundary/clock/testkit"
	boundary "go-boilerplate/internal/usecase/boundary/exchangerate"
	"go-boilerplate/pkg/decimal"

	"github.com/stretchr/testify/assert"
)

func Test_cacheGateway_store(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	rate := &boundary.Rate{Base: "USD", Quote: "JPY", Value: decimal.FromInt(1)}

	newCG := func() *cacheGateway {
		return &cacheGateway{
			clk:   testkit.NewStepClock(start, 0),
			rates: make(map[cacheKey]cacheEntry),
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限内はすべて保存される", func(t *testing.T) {
			t.Parallel()
			cg := newCG()
			for i := range maxCacheEntries {
				cg.store(cacheKey{base: "USD", quote: strconv.Itoa(i)}, rate)
			}
			assert.Len(t, cg.rates, maxCacheEntries)
		})
	})

	t.Run("境界ケース", func(t *testing.T) {
		t.Parallel()

		t.Run("上限到達かつ全て有効なら新規は保存しない", func(t *testing.T) {
			t.Parallel()
			cg := newCG()
			for i := range maxCacheEntries {
				cg.rates[cacheKey{base: "USD", quote: strconv.Itoa(i)}] = cacheEntry{rate: rate, expiresAt: start.Add(rateTTL)}
			}
			cg.store(cacheKey{base: "USD", quote: "new"}, rate)

			assert.Len(t, cg.rates, maxCacheEntries)
			_, ok := cg.rates[cacheKey{base: "USD", quote: "new"}]
			assert.False(t, ok)
		})

		t.Run("上限到達でも失効エントリを掃除して保存する", func(t *testing.T) {
			t.Parallel()
			cg := newCG()
			for i := range maxCacheEntries {
				cg.rates[cacheKey{base: "USD", quote: strconv.Itoa(i)}] = cacheEntry{rate: rate, expiresAt: start.Add(-time.Hour)}
			}
			cg.store(cacheKey{base: "USD", quote: "new"}, rate)

			_, ok := cg.rates[cacheKey{base: "USD", quote: "new"}]
			assert.True(t, ok)
		})
	})
}
