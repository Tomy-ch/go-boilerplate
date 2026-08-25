package publisher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/publisher"
)

// newTestPublisher は、判別子に対応する publish 実装を構築します。
// New の引数はサンプル削除で変わるため、呼び出しをここへ集約してマーカーを 1 箇所に留めます。
func newTestPublisher(
	t *testing.T, cfg *config.OutboxConfig, epCfg *config.EndpointConfig,
) (boundary.Publisher, error) {
	t.Helper()
	return New(cfg, epCfg, nil, observability.NewNoopTracerFactory(t))
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
	})
}
