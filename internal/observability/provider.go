package observability

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/system"
	"go-boilerplate/pkg/xerrors"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// NewResource は service 識別情報を付与した OpenTelemetry リソースを生成する。
func NewResource(appCfg *config.ApplicationConfig, bi system.BuildInfo) (*resource.Resource, error) {
	attrs := resource.NewSchemaless(
		semconv.ServiceName(appCfg.Name()),
		semconv.DeploymentEnvironmentName(appCfg.Env()),
		semconv.ServiceVersion(bi.Version()),
		attribute.String("service.revision", bi.Revision()),
		attribute.String("service.build_date", bi.BuildDate()),
	)

	res, err := resource.Merge(resource.Default(), attrs)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to merge otel resource")
	}

	return res, nil
}

// NewTracerProvider は TracerProvider と W3C 伝播器をグローバル登録し、構築した TracerProvider を返す。
// SpanExporter は標準 OTEL_* env から構築し、送出先未指定時は no-op となる。
func NewTracerProvider(res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := autoexport.NewSpanExporter(context.Background(), autoexport.WithFallbackSpanExporter(newNoopSpanExporter))
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to build span exporter")
	}

	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if !isNoopSpanExporter(exporter) {
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	tp := sdktrace.NewTracerProvider(opts...)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(NewTextMapPropagator())

	return tp, nil
}

// NewTextMapPropagator は、W3C TraceContext + Baggage の複合 propagator を返します。
// inbound/DB のグローバル設定と outbound substrate への注入で同一インスタンスを共有します。
func NewTextMapPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// NewMeterProvider は MeterProvider をグローバル登録し、構築した MeterProvider を返す。
// MetricReader は標準 OTEL_* env から構築し、送出先未指定時は no-op となりランタイム計装も行わない。
func NewMeterProvider(res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	reader, err := autoexport.NewMetricReader(context.Background(), autoexport.WithFallbackMetricReader(newNoopMetricReader))
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to build metric reader")
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)

	if !isNoopMetricReader(reader) {
		if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
			return nil, xerrors.Wrap(err, "failed to start runtime metrics")
		}
	}

	otel.SetMeterProvider(mp)

	return mp, nil
}

// ProvideTracerProvider は具象 TracerProvider を otel の trace.TracerProvider IF として返す。
func ProvideTracerProvider(tp *sdktrace.TracerProvider) trace.TracerProvider { return tp }

// ProvideMeterProvider は具象 MeterProvider を otel の metric.MeterProvider IF として返す。
// otel 型を DI 層へ漏らさず WorkerMetrics 等へ MeterProvider を注入するための変換点。
func ProvideMeterProvider(mp *sdkmetric.MeterProvider) metric.MeterProvider { return mp }
