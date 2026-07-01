package module

import (
	"testing"

	"go-boilerplate/internal/config"
	outboxuc "go-boilerplate/internal/usecase/outbox"

	"github.com/stretchr/testify/assert"
)

func Test_provideRelaySettings(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("BatchSize が正なら設定値をそのまま使う", func(t *testing.T) {
			t.Parallel()
			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxBatchSize(t, 50)

			got := provideRelaySettings(cfg)

			assert.Equal(t, int32(50), got.BatchSize)
			assert.Equal(t, cfg.PollInterval(), got.PollInterval)
			assert.Equal(t, cfg.ErrorBackoff(), got.ErrorBackoff)
		})

		t.Run("BatchSize が 0 なら DefaultBatchSize に clamp する", func(t *testing.T) {
			t.Parallel()
			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxBatchSize(t, 0)

			got := provideRelaySettings(cfg)

			assert.Equal(t, outboxuc.DefaultBatchSize, got.BatchSize)
			assert.Equal(t, cfg.PollInterval(), got.PollInterval)
			assert.Equal(t, cfg.ErrorBackoff(), got.ErrorBackoff)
		})

		t.Run("BatchSize が負なら DefaultBatchSize に clamp する", func(t *testing.T) {
			t.Parallel()
			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxBatchSize(t, -1)

			got := provideRelaySettings(cfg)

			assert.Equal(t, outboxuc.DefaultBatchSize, got.BatchSize)
			assert.Equal(t, cfg.PollInterval(), got.PollInterval)
			assert.Equal(t, cfg.ErrorBackoff(), got.ErrorBackoff)
		})
	})
}
