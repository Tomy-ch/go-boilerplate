package publisher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
)

func Test_newQueueConfig(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("queue URL と region が揃っていれば adapter 設定を返す", func(t *testing.T) {
			t.Parallel()

			const queueURL = "http://elasticmq:9324/000000000000/gobp-events"
			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxQueue(t, queueURL, "us-east-1", "k", "s")

			got, err := newQueueConfig(cfg)

			require.NoError(t, err)
			assert.Equal(t, queueURL, got.QueueURL)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("queue URL が空なら起動時点で弾く", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxQueue(t, "", "us-east-1", "k", "s")

			_, err := newQueueConfig(cfg)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidQueue)
		})

		t.Run("資格情報が空なら起動時点で弾く", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxQueue(t, "http://elasticmq:9324/000000000000/gobp-events", "us-east-1", "", "")

			_, err := newQueueConfig(cfg)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidQueue)
		})

		t.Run("region が空なら起動時点で弾く", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxQueue(t, "http://elasticmq:9324/000000000000/gobp-events", "", "k", "s")

			_, err := newQueueConfig(cfg)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidQueue)
		})
	})
}
