package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
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

		t.Run("公開関数はグローバル伝播器で attrs の traceparent から trace を復元する", func(t *testing.T) {
			t.Parallel()

			// 公開関数は otel.GetTextMapPropagator() を参照するため、TraceContext を一時登録する。
			prev := otel.GetTextMapPropagator()
			otel.SetTextMapPropagator(propagation.TraceContext{})
			t.Cleanup(func() { otel.SetTextMapPropagator(prev) })

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
			carrier := propagation.MapCarrier{}
			propagation.TraceContext{}.Inject(trace.ContextWithSpanContext(context.Background(), sc), carrier)

			got := ExtractFromCarrier(context.Background(), map[string]string(carrier))

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

		t.Run("公開関数は TraceContext 限定の伝播器を用いる", func(t *testing.T) {
			t.Parallel()

			// グローバル伝播器に依らず TraceContext 限定で inject する公開経路を通す。
			assert.NotPanics(t, func() { InjectTraceContextToCarrier(context.Background(), map[string]string{}) })
		})

		t.Run("公開関数は trace context を inject しつつ baggage は転送しない", func(t *testing.T) {
			t.Parallel()

			// span が無いと traceparent も書かれず NotContains が疑似陽性になるため、
			// span 有りで traceparent が書かれた上で baggage だけが落ちることを検証する。
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

			member, err := baggage.NewMember("tenant", "acme")
			require.NoError(t, err)
			bag, err := baggage.New(member)
			require.NoError(t, err)
			ctx = baggage.ContextWithBaggage(ctx, bag)

			attrs := map[string]string{}
			InjectTraceContextToCarrier(ctx, attrs)

			// traceparent は伝搬され、baggage は TraceContext 限定の公開関数によって転送されない。
			assert.Contains(t, attrs, "traceparent")
			assert.NotContains(t, attrs, "baggage")
		})
	})
}
