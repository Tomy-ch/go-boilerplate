package instrumentation

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObservabilityMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("可観測が有効ならotelechoミドルウェアを提供する", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			obsCfg := config.NewObservabilityConfig(cfg)

			mw := ObservabilityMiddleware(appCfg, obsCfg)

			assert.Equal(t, observabilityPriority, mw.Middleware.Priority)
			require.NotNil(t, mw.Middleware.Middleware)
		})

		t.Run("可観測が無効でも素通しミドルウェアを提供する", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			obsCfg := config.NewObservabilityConfig(cfg)
			obsCfg.SetObservabilityTracesExporter(t, "")
			obsCfg.SetObservabilityMetricsExporter(t, "")

			mw := ObservabilityMiddleware(appCfg, obsCfg)

			assert.Equal(t, observabilityPriority, mw.Middleware.Priority)
			require.NotNil(t, mw.Middleware.Middleware)
		})
	})
}
