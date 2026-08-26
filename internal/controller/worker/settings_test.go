package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/pkg/backoff"
)

func TestSettings_circuitBackoff(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Open の cooldown 設定と既定倍率を持つ算出器を返す", func(t *testing.T) {
			t.Parallel()

			s := Settings{CircuitOpenBackoffInitial: 2 * time.Second, CircuitOpenBackoffMax: 20 * time.Second}

			assert.Equal(t, backoff.Exponential{
				Initial:    2 * time.Second,
				Max:        20 * time.Second,
				Multiplier: circuitBackoffMultiplier,
			}, s.circuitBackoff())
		})
	})
}

func TestSettings_nackBackoff(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("per-message 再配送 backoff の設定と既定倍率を持つ算出器を返す", func(t *testing.T) {
			t.Parallel()

			s := Settings{NackBackoffInitial: 3 * time.Second, NackBackoffMax: 30 * time.Second}

			assert.Equal(t, backoff.Exponential{
				Initial:    3 * time.Second,
				Max:        30 * time.Second,
				Multiplier: nackBackoffMultiplier,
			}, s.nackBackoff())
		})
	})
}
