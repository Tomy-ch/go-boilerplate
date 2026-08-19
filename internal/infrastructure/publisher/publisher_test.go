package publisher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/queue/sqs" // sample-api:line
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/publisher"
)

// newTestPublisher は、判別子に対応する publish 実装を構築します。
// New の引数はサンプル削除で変わるため、呼び出しをここへ集約してマーカーを 1 箇所に留めます。
func newTestPublisher(
	t *testing.T, cfg *config.OutboxConfig, epCfg *config.EndpointConfig,
) (boundary.Publisher, error) {
	t.Helper()
	// sample-api:replace-begin
	return New(cfg, epCfg, nil, observability.NewDisabledOutboundHTTPClient(true), observability.NewNoopTracerFactory(t))
	// sample-api:replace-with
	// = return New(cfg, epCfg, nil, observability.NewNoopTracerFactory(t))
	// sample-api:replace-end
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("判別子が http なら HTTP 実装を返す", func(t *testing.T) {
			t.Parallel()

			mock := config.MockConfigForTest(t)
			cfg := config.NewOutboxConfig(mock)
			epCfg := config.NewEndpointConfig(mock)
			cfg.SetOutboxPublisher(t, KindHTTP)
			epCfg.SetEndpointOutbox(t, "https://example.com/ingest")

			got, err := newTestPublisher(t, cfg, epCfg)

			require.NoError(t, err)
			assert.IsType(t, &httpPublisher{}, got)
		})

		// sample-api:begin
		t.Run("判別子が sqs なら SQS 実装を返す", func(t *testing.T) {
			t.Parallel()

			mock := config.MockConfigForTest(t)
			cfg := config.NewOutboxConfig(mock)
			epCfg := config.NewEndpointConfig(mock)
			cfg.SetOutboxPublisher(t, KindSQS)
			cfg.SetOutboxQueue(t, "http://elasticmq:9324/000000000000/gobp-events", "us-east-1", "k", "s")

			got, err := newTestPublisher(t, cfg, epCfg)

			require.NoError(t, err)
			// SQS 実装は別パッケージの非公開型のため、同型のインスタンスを渡して型を固定する。
			// 「HTTP でない」という消去法だと、分岐が増えたときに取り違えを検知できない。
			assert.IsType(t, sqs.NewPublisher(nil, sqs.PublisherConfig{}, observability.NewNoopTracerFactory(t)), got)
		})

		t.Run("判別子が sqs なら ENDPOINT_OUTBOX を要求しない", func(t *testing.T) {
			t.Parallel()

			mock := config.MockConfigForTest(t)
			cfg := config.NewOutboxConfig(mock)
			epCfg := config.NewEndpointConfig(mock)
			cfg.SetOutboxPublisher(t, KindSQS)
			epCfg.SetEndpointOutbox(t, "")
			cfg.SetOutboxQueue(t, "http://elasticmq:9324/000000000000/gobp-events", "us-east-1", "k", "s")

			_, err := newTestPublisher(t, cfg, epCfg)

			require.NoError(t, err)
		})
		// sample-api:end
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知の判別子は起動エラーにする", func(t *testing.T) {
			t.Parallel()

			mock := config.MockConfigForTest(t)
			cfg := config.NewOutboxConfig(mock)
			epCfg := config.NewEndpointConfig(mock)
			cfg.SetOutboxPublisher(t, "kafka")

			_, err := newTestPublisher(t, cfg, epCfg)

			require.ErrorIs(t, err, ErrUnknownKind)
		})

		t.Run("判別子が空でも既定へ流さず起動エラーにする", func(t *testing.T) {
			t.Parallel()

			mock := config.MockConfigForTest(t)
			cfg := config.NewOutboxConfig(mock)
			epCfg := config.NewEndpointConfig(mock)
			cfg.SetOutboxPublisher(t, "")

			_, err := newTestPublisher(t, cfg, epCfg)

			require.ErrorIs(t, err, ErrUnknownKind)
		})

		t.Run("http なのに送信先が未設定なら起動エラーにする", func(t *testing.T) {
			t.Parallel()

			mock := config.MockConfigForTest(t)
			cfg := config.NewOutboxConfig(mock)
			epCfg := config.NewEndpointConfig(mock)
			cfg.SetOutboxPublisher(t, KindHTTP)
			epCfg.SetEndpointOutbox(t, "")

			_, err := newTestPublisher(t, cfg, epCfg)

			require.ErrorIs(t, err, ErrInvalidEndpoint)
		})

		// sample-api:begin
		t.Run("sqs なのに queue URL が未設定なら起動エラーにする", func(t *testing.T) {
			t.Parallel()

			mock := config.MockConfigForTest(t)
			cfg := config.NewOutboxConfig(mock)
			epCfg := config.NewEndpointConfig(mock)
			cfg.SetOutboxPublisher(t, KindSQS)
			cfg.SetOutboxQueue(t, "", "us-east-1", "k", "s")

			_, err := newTestPublisher(t, cfg, epCfg)

			require.ErrorIs(t, err, ErrInvalidQueue)
		})
		// sample-api:end
	})
}
