package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func Test_extractFromCarrier_D1(t *testing.T) {
	t.Parallel()

	prop := propagation.TraceContext{}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("attrs の traceparent から trace context を継続する", func(t *testing.T) {
			t.Parallel()

			traceID, err := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
			require.NoError(t, err)
			spanID, err := trace.SpanIDFromHex("0123456789abcdef")
			require.NoError(t, err)
			sc := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
				Remote:     true,
			})
			// producer 側が attrs に traceparent を載せた状態を作る
			carrier := propagation.MapCarrier{}
			prop.Inject(trace.ContextWithSpanContext(context.Background(), sc), carrier)

			got := extractFromCarrier(context.Background(), map[string]string(carrier), prop)

			gsc := trace.SpanContextFromContext(got)
			require.True(t, gsc.HasTraceID())
			assert.Equal(t, traceID, gsc.TraceID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("attrs が空なら trace context は付与されない", func(t *testing.T) {
			t.Parallel()

			got := extractFromCarrier(context.Background(), nil, prop)

			assert.False(t, trace.SpanContextFromContext(got).HasTraceID())
		})

		t.Run("公開関数はグローバル伝播器を用い、attrs が空なら ctx をそのまま返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// グローバル伝播器を変更せず公開経路（otel.GetTextMapPropagator 利用）を通す。
			assert.Equal(t, ctx, ExtractFromCarrier(ctx, nil))
		})
	})
}

func Test_injectToCarrier_D1(t *testing.T) {
	t.Parallel()

	prop := propagation.TraceContext{}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctx の trace context を attrs へ traceparent として書き込む", func(t *testing.T) {
			t.Parallel()

			traceID, err := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
			require.NoError(t, err)
			spanID, err := trace.SpanIDFromHex("0123456789abcdef")
			require.NoError(t, err)
			sc := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			})
			ctx := trace.ContextWithSpanContext(context.Background(), sc)

			attrs := map[string]string{}
			injectToCarrier(ctx, attrs, prop)

			require.Contains(t, attrs, "traceparent")
			// inject した attrs から復元すると同一 traceID に繋がる（往復確認）。
			got := extractFromCarrier(context.Background(), attrs, prop)
			assert.Equal(t, traceID, trace.SpanContextFromContext(got).TraceID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("attrs が nil なら何もしない", func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() { injectToCarrier(context.Background(), nil, prop) })
		})

		t.Run("公開関数はグローバル伝播器を用いる", func(t *testing.T) {
			t.Parallel()

			// グローバル伝播器を変更せず公開経路（otel.GetTextMapPropagator 利用）を通す。
			assert.NotPanics(t, func() { InjectToCarrier(context.Background(), map[string]string{}) })
		})
	})
}
