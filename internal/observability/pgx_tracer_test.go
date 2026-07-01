package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewPgxTracer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("provider を注入した otelpgx トレーサーを返す", func(t *testing.T) {
			t.Parallel()

			tracer := NewPgxTracer(noop.NewTracerProvider(), sdkmetric.NewMeterProvider())
			assert.NotNil(t, tracer)
		})
	})
}
