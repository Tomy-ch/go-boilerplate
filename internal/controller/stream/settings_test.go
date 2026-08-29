package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSettings_withDefaults(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ゼロ値は既定値に寄る", func(t *testing.T) {
			t.Parallel()

			got := Settings{}.withDefaults()

			assert.Equal(t, DefaultMaxConnections, got.MaxConnections)
			assert.Equal(t, DefaultReplayConcurrency, got.ReplayConcurrency)
		})

		t.Run("負の値も既定値に寄る", func(t *testing.T) {
			t.Parallel()

			got := Settings{MaxConnections: -1, ReplayConcurrency: -1}.withDefaults()

			assert.Equal(t, DefaultMaxConnections, got.MaxConnections)
			assert.Equal(t, DefaultReplayConcurrency, got.ReplayConcurrency)
		})

		t.Run("設定された値はそのまま残る", func(t *testing.T) {
			t.Parallel()

			got := Settings{MaxConnections: 3, ReplayConcurrency: 1}.withDefaults()

			assert.Equal(t, 3, got.MaxConnections)
			assert.Equal(t, 1, got.ReplayConcurrency)
		})
	})
}
