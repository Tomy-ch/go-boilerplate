package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestSSRFGuardControl(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("通常のグローバルアドレスは許可する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ssrfGuardControl("tcp", "93.184.216.34:443", nil))
		})

		t.Run("ループバックは許可する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ssrfGuardControl("tcp", "127.0.0.1:8080", nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クラウドメタデータ等のリンクローカルは拒否する", func(t *testing.T) {
			t.Parallel()
			require.Error(t, ssrfGuardControl("tcp", "169.254.169.254:80", nil))
		})
	})
}

func newSampledContext() context.Context {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		SpanID:     trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.FlagsSampled,
	})
	return trace.ContextWithSpanContext(context.Background(), sc)
}

func TestConditionalPropagator(t *testing.T) {
	t.Parallel()

	prop := conditionalPropagator{inner: propagation.TraceContext{}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フラグ未設定なら通常どおり伝搬する", func(t *testing.T) {
			t.Parallel()

			carrier := propagation.MapCarrier{}
			prop.Inject(newSampledContext(), carrier)
			assert.NotEmpty(t, carrier.Get("traceparent"))
		})

		t.Run("伝搬有効フラグでは伝搬する", func(t *testing.T) {
			t.Parallel()

			ctx := ContextWithTracePropagation(newSampledContext(), true)
			carrier := propagation.MapCarrier{}
			prop.Inject(ctx, carrier)
			assert.NotEmpty(t, carrier.Get("traceparent"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("伝搬無効フラグでは外部へ注入しない", func(t *testing.T) {
			t.Parallel()

			ctx := ContextWithTracePropagation(newSampledContext(), false)
			carrier := propagation.MapCarrier{}
			prop.Inject(ctx, carrier)
			assert.Empty(t, carrier.Get("traceparent"))
		})
	})
}
