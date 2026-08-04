package publisher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/queue/sqs"
	"go-boilerplate/internal/observability"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("判別子が http なら HTTP 実装を返す", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxPublisher(t, KindHTTP)
			cfg.SetOutboxEndpoint(t, "https://example.com/ingest")

			got, err := New(cfg, nil, observability.NewNoopTracerFactory(t))

			require.NoError(t, err)
			assert.IsType(t, &httpPublisher{}, got)
		})

		t.Run("判別子が sqs なら SQS 実装を返す", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxPublisher(t, KindSQS)
			cfg.SetOutboxQueue(t, "http://elasticmq:9324/000000000000/gobp-events", "us-east-1", "k", "s")

			got, err := New(cfg, nil, observability.NewNoopTracerFactory(t))

			require.NoError(t, err)
			// SQS 実装は別パッケージの非公開型のため、同型のインスタンスを渡して型を固定する。
			// 「HTTP でない」という消去法だと、分岐が増えたときに取り違えを検知できない。
			assert.IsType(t, sqs.NewPublisher(nil, sqs.PublisherConfig{}, observability.NewNoopTracerFactory(t)), got)
		})

		t.Run("判別子が sqs なら OUTBOX_ENDPOINT を要求しない", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxPublisher(t, KindSQS)
			cfg.SetOutboxEndpoint(t, "")
			cfg.SetOutboxQueue(t, "http://elasticmq:9324/000000000000/gobp-events", "us-east-1", "k", "s")

			_, err := New(cfg, nil, observability.NewNoopTracerFactory(t))

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知の判別子は起動エラーにする", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxPublisher(t, "kafka")

			_, err := New(cfg, nil, observability.NewNoopTracerFactory(t))

			require.ErrorIs(t, err, ErrUnknownKind)
		})

		t.Run("判別子が空でも既定へ流さず起動エラーにする", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxPublisher(t, "")

			_, err := New(cfg, nil, observability.NewNoopTracerFactory(t))

			require.ErrorIs(t, err, ErrUnknownKind)
		})

		t.Run("http なのに送信先が未設定なら起動エラーにする", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxPublisher(t, KindHTTP)
			cfg.SetOutboxEndpoint(t, "")

			_, err := New(cfg, nil, observability.NewNoopTracerFactory(t))

			require.ErrorIs(t, err, ErrInvalidEndpoint)
		})

		t.Run("sqs なのに queue URL が未設定なら起動エラーにする", func(t *testing.T) {
			t.Parallel()

			cfg := config.NewOutboxConfig(config.MockConfigForTest(t))
			cfg.SetOutboxPublisher(t, KindSQS)
			cfg.SetOutboxQueue(t, "", "us-east-1", "k", "s")

			_, err := New(cfg, nil, observability.NewNoopTracerFactory(t))

			require.ErrorIs(t, err, ErrInvalidQueue)
		})
	})
}
