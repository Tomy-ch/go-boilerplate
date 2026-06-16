package observability

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/system"
	"go-boilerplate/pkg/xerrors"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// noopSpanExporter は、span を送出しない SpanExporter です。
type noopSpanExporter struct{}

// NewResource は、サービス識別情報を付与した OpenTelemetry リソースを生成します。
// service.name / deployment.environment は既存のアプリ設定、service.version はビルド時注入(ldflags)に
// 由来し、OTel 固有の env / typed config は経由しません。
func NewResource(appCfg *config.ApplicationConfig, bi system.BuildInfo) *resource.Resource {
	attrs := resource.NewSchemaless(
		semconv.ServiceName(appCfg.Name()),
		semconv.DeploymentEnvironmentName(appCfg.Env()),
		semconv.ServiceVersion(bi.Version()),
	)

	res, err := resource.Merge(resource.Default(), attrs)
	if err != nil {
		// schema URL 不一致等で部分マージに失敗した場合は、サービス識別属性を保持する側を採用する。
		return attrs
	}

	return res
}

// TracerProvider は、TracerProvider を初期化してグローバル登録し、W3C TraceContext + Baggage の
// 伝播器を設定したうえで、Shutdown フックを Registrar へ登録して返します。
// SpanExporter は標準 env(OTEL_TRACES_EXPORTER / OTEL_EXPORTER_OTLP_*)から構築し、未設定時は送出しない
// no-op にフォールバックします。サンプリングは OTEL_TRACES_SAMPLER に従います(未指定時は親準拠の常時採取)。
func TracerProvider(reg lifecycle.Registrar, res *resource.Resource) (trace.TracerProvider, error) {
	ctx := context.Background()

	exporter, err := autoexport.NewSpanExporter(ctx, autoexport.WithFallbackSpanExporter(newNoopSpanExporter))
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to build span exporter")
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	reg.RegisterStop(tp.Shutdown)

	return tp, nil
}

// MeterProvider は、MeterProvider を初期化してグローバル登録し、Go ランタイムメトリクスの計装を開始した
// うえで、Shutdown フックを Registrar へ登録して返します。MetricReader は標準
// env(OTEL_METRICS_EXPORTER / OTEL_EXPORTER_OTLP_*)から構築し、未設定時は送出しない no-op にフォールバック
// します。
func MeterProvider(reg lifecycle.Registrar, res *resource.Resource) (metric.MeterProvider, error) {
	ctx := context.Background()

	reader, err := autoexport.NewMetricReader(ctx, autoexport.WithFallbackMetricReader(newNoopMetricReader))
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to build metric reader")
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)

	otel.SetMeterProvider(mp)
	reg.RegisterStop(mp.Shutdown)

	if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
		return nil, xerrors.Wrap(err, "failed to start runtime metrics")
	}

	return mp, nil
}

// InvokeMeterProvider は、MeterProvider を fx に明示的に構築させるための no-op invoke target です。
// MeterProvider には他に依存元が無いため、グローバル登録とランタイムメトリクス計装を確実に起動させます。
func InvokeMeterProvider(metric.MeterProvider) {}

// newNoopSpanExporter は、OTEL_TRACES_EXPORTER 未設定時のフォールバック SpanExporter を返します。
func newNoopSpanExporter(context.Context) (sdktrace.SpanExporter, error) {
	return noopSpanExporter{}, nil
}

func (noopSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }

func (noopSpanExporter) Shutdown(context.Context) error { return nil }

// newNoopMetricReader は、OTEL_METRICS_EXPORTER 未設定時のフォールバック MetricReader を返します。
// ManualReader は周期送出も外部接続も行わないため、実質的に no-op として機能します。
func newNoopMetricReader(context.Context) (sdkmetric.Reader, error) {
	return sdkmetric.NewManualReader(), nil
}
