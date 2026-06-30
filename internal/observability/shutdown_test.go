package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// errSpanExporter は Shutdown で任意のエラーを返す SpanExporter スタブ。
// TracerProvider.Shutdown 経由でエラーを発生させ、結合エラーの伝播を検証するために使う。
type errSpanExporter struct{ err error }

// errMetricExporter は Shutdown で任意のエラーを返す metric Exporter スタブ。
// PeriodicReader 経由で MeterProvider.Shutdown にエラーを発生させ、結合エラーの伝播を検証するために使う。
type errMetricExporter struct{ err error }

// errLogExporter は Shutdown で任意のエラーを返す log Exporter スタブ。
// BatchProcessor 経由で LoggerProvider.Shutdown にエラーを発生させ、結合エラーの伝播を検証するために使う。
type errLogExporter struct{ err error }

func (errSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }

func (e errSpanExporter) Shutdown(context.Context) error { return e.err }

func (errMetricExporter) Temporality(k sdkmetric.InstrumentKind) metricdata.Temporality {
	return sdkmetric.DefaultTemporalitySelector(k)
}

func (errMetricExporter) Aggregation(k sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(k)
}

func (errMetricExporter) Export(context.Context, *metricdata.ResourceMetrics) error { return nil }

func (errMetricExporter) ForceFlush(context.Context) error { return nil }

func (e errMetricExporter) Shutdown(context.Context) error { return e.err }

func (errLogExporter) Export(context.Context, []sdklog.Record) error { return nil }

func (e errLogExporter) Shutdown(context.Context) error { return e.err }

func (errLogExporter) ForceFlush(context.Context) error { return nil }

func TestNewProviderShutdowner(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("具象プロバイダを束ねた ProviderShutdowner を返す", func(t *testing.T) {
			t.Parallel()

			s := NewProviderShutdowner(
				sdktrace.NewTracerProvider(), sdkmetric.NewMeterProvider(), sdklog.NewLoggerProvider(),
			)

			require.NotNil(t, s)
		})
	})
}

func TestProviderShutdowner_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("TracerProvider/MeterProvider/LoggerProviderをShutdownしエラーを返さない", func(t *testing.T) {
			t.Parallel()

			s := NewProviderShutdowner(
				sdktrace.NewTracerProvider(), sdkmetric.NewMeterProvider(), sdklog.NewLoggerProvider(),
			)

			require.NoError(t, s.Shutdown(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("TracerProviderのShutdownが失敗した場合は結合エラーとして伝播する", func(t *testing.T) {
			t.Parallel()

			wantErr := errors.New("span exporter shutdown failed")
			// バッチャ経由で失敗する exporter を仕込み、tp.Shutdown がエラーを返すようにする。
			tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(errSpanExporter{err: wantErr}))
			mp := sdkmetric.NewMeterProvider()
			lp := sdklog.NewLoggerProvider()

			// errors.Join は全引数（tp/mp/lp の Shutdown）を評価してから結合するため、
			// tp が失敗しても mp / lp の Shutdown は必ず呼ばれる（Go の引数評価保証）。
			err := NewProviderShutdowner(tp, mp, lp).Shutdown(context.Background())

			require.ErrorIs(t, err, wantErr)
		})

		t.Run("MeterProviderのShutdownが失敗した場合は結合エラーとして伝播する", func(t *testing.T) {
			t.Parallel()

			wantErr := errors.New("metric exporter shutdown failed")
			// PeriodicReader 経由で失敗する exporter を仕込み、mp.Shutdown がエラーを返すようにする。
			mp := sdkmetric.NewMeterProvider(
				sdkmetric.WithReader(sdkmetric.NewPeriodicReader(errMetricExporter{err: wantErr})),
			)
			tp := sdktrace.NewTracerProvider()
			lp := sdklog.NewLoggerProvider()

			err := NewProviderShutdowner(tp, mp, lp).Shutdown(context.Background())

			require.ErrorIs(t, err, wantErr)
		})

		t.Run("LoggerProviderのShutdownが失敗した場合は結合エラーとして伝播する", func(t *testing.T) {
			t.Parallel()

			wantErr := errors.New("log exporter shutdown failed")
			// BatchProcessor 経由で失敗する exporter を仕込み、lp.Shutdown がエラーを返すようにする。
			lp := sdklog.NewLoggerProvider(
				sdklog.WithProcessor(sdklog.NewBatchProcessor(errLogExporter{err: wantErr})),
			)
			tp := sdktrace.NewTracerProvider()
			mp := sdkmetric.NewMeterProvider()

			err := NewProviderShutdowner(tp, mp, lp).Shutdown(context.Background())

			require.ErrorIs(t, err, wantErr)
		})

		t.Run("全Providerの失敗をerrors.Joinで集約して伝播する", func(t *testing.T) {
			t.Parallel()

			tpErr := errors.New("span exporter shutdown failed")
			mpErr := errors.New("metric exporter shutdown failed")
			lpErr := errors.New("log exporter shutdown failed")

			tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(errSpanExporter{err: tpErr}))
			mp := sdkmetric.NewMeterProvider(
				sdkmetric.WithReader(sdkmetric.NewPeriodicReader(errMetricExporter{err: mpErr})),
			)
			lp := sdklog.NewLoggerProvider(
				sdklog.WithProcessor(sdklog.NewBatchProcessor(errLogExporter{err: lpErr})),
			)

			err := NewProviderShutdowner(tp, mp, lp).Shutdown(context.Background())

			// errors.Join により tp/mp/lp 全ての Shutdown エラーが 1 つに集約されることを確認する。
			require.ErrorIs(t, err, tpErr)
			require.ErrorIs(t, err, mpErr)
			require.ErrorIs(t, err, lpErr)
		})
	})
}
