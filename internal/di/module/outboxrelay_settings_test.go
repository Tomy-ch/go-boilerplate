package module

import (
	"testing"
	"time"

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

		t.Run("PollInterval が正なら設定値をそのまま使う", func(t *testing.T) {
			t.Parallel()
			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxPollInterval(t, 3*time.Second)

			got := provideRelaySettings(cfg)

			assert.Equal(t, 3*time.Second, got.PollInterval)
		})

		t.Run("PollInterval が 0 なら既定値に clamp する", func(t *testing.T) {
			t.Parallel()
			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxPollInterval(t, 0)

			got := provideRelaySettings(cfg)

			assert.Equal(t, defaultRelayPollInterval, got.PollInterval)
		})

		t.Run("PollInterval が負なら既定値に clamp する", func(t *testing.T) {
			t.Parallel()
			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxPollInterval(t, -1)

			got := provideRelaySettings(cfg)

			assert.Equal(t, defaultRelayPollInterval, got.PollInterval)
		})

		t.Run("ErrorBackoff が正なら設定値をそのまま使う", func(t *testing.T) {
			t.Parallel()
			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxErrorBackoff(t, 7*time.Second)

			got := provideRelaySettings(cfg)

			assert.Equal(t, 7*time.Second, got.ErrorBackoff)
		})

		t.Run("ErrorBackoff が 0 なら既定値に clamp する", func(t *testing.T) {
			t.Parallel()
			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxErrorBackoff(t, 0)

			got := provideRelaySettings(cfg)

			assert.Equal(t, defaultRelayErrorBackoff, got.ErrorBackoff)
		})

		t.Run("ErrorBackoff が負なら既定値に clamp する", func(t *testing.T) {
			t.Parallel()
			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxErrorBackoff(t, -1)

			got := provideRelaySettings(cfg)

			assert.Equal(t, defaultRelayErrorBackoff, got.ErrorBackoff)
		})
	})
}
