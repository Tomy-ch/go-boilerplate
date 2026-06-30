package instrumentation

import (
	"reflect"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
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

			// otelecho ミドルウェアは next を別ハンドラでラップするため、
			// 適用結果は元のハンドラと別関数になる（素通しとの挙動差を識別する）。
			dummy := echo.HandlerFunc(func(echo.Context) error { return nil })
			wrapped := mw.Middleware.Middleware(dummy)
			assert.NotEqual(t, reflect.ValueOf(dummy).Pointer(), reflect.ValueOf(wrapped).Pointer())
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

			// 素通しミドルウェアは next をそのまま返す性質を持つため、
			// 適用結果は元のハンドラと同一関数になる（有効ケースとの挙動差を識別する）。
			dummy := echo.HandlerFunc(func(echo.Context) error { return nil })
			wrapped := mw.Middleware.Middleware(dummy)
			assert.Equal(t, reflect.ValueOf(dummy).Pointer(), reflect.ValueOf(wrapped).Pointer())
		})
	})
}
