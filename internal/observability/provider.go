package observability

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/system"
	"go-boilerplate/pkg/xerrors"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// OTLP プロトコル指定（OBS_OTLP_PROTOCOL）。
const (
	protocolHTTP = "http/protobuf"
	protocolGRPC = "grpc"
)

// errInvalidOTLPProtocol は、OBS_OTLP_PROTOCOL が http/protobuf でも grpc でもない場合に返されます。
var errInvalidOTLPProtocol = xerrors.New("invalid OTLP protocol (want http/protobuf or grpc)")

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
func NewTracerProvider(obsCfg *config.ObservabilityConfig, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}

	if obsCfg.TracesEnabled() {
		exporter, err := newSpanExporter(context.Background(), obsCfg)
		if err != nil {
			return nil, xerrors.Wrap(err, "failed to build span exporter")
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	tp := sdktrace.NewTracerProvider(opts...)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// NewMeterProvider は MeterProvider をグローバル登録し、構築した MeterProvider を返す。
func NewMeterProvider(obsCfg *config.ObservabilityConfig, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	opts := []sdkmetric.Option{sdkmetric.WithResource(res)}

	if obsCfg.MetricsEnabled() {
		reader, err := newMetricReader(context.Background(), obsCfg)
		if err != nil {
			return nil, xerrors.Wrap(err, "failed to build metric reader")
		}
		opts = append(opts, sdkmetric.WithReader(reader))
	}

	mp := sdkmetric.NewMeterProvider(opts...)

	if obsCfg.MetricsEnabled() {
		if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
			return nil, xerrors.Wrap(err, "failed to start runtime metrics")
		}
	}

	otel.SetMeterProvider(mp)

	return mp, nil
}

// newSpanExporter は OBS_OTLP_PROTOCOL / OBS_OTLP_ENDPOINT から OTLP SpanExporter を構築する。
// endpoint 未指定時は OTLP のデフォルト（localhost:4318 / :4317）に従う。
func newSpanExporter(ctx context.Context, obsCfg *config.ObservabilityConfig) (sdktrace.SpanExporter, error) {
	switch obsCfg.OTLPProtocol() {
	case protocolGRPC:
		var opts []otlptracegrpc.Option
		if ep := obsCfg.OTLPEndpoint(); ep != "" {
			opts = append(opts, otlptracegrpc.WithEndpointURL(ep))
		}
		return otlptracegrpc.New(ctx, opts...)
	case protocolHTTP, "":
		var opts []otlptracehttp.Option
		if ep := obsCfg.OTLPEndpoint(); ep != "" {
			opts = append(opts, otlptracehttp.WithEndpointURL(ep))
		}
		return otlptracehttp.New(ctx, opts...)
	default:
		return nil, errInvalidOTLPProtocol
	}
}

// newMetricReader は OTLP MetricExporter を PeriodicReader で包んで返す。
func newMetricReader(ctx context.Context, obsCfg *config.ObservabilityConfig) (sdkmetric.Reader, error) {
	exporter, err := newMetricExporter(ctx, obsCfg)
	if err != nil {
		return nil, err
	}
	return sdkmetric.NewPeriodicReader(exporter), nil
}

// newMetricExporter は OBS_OTLP_PROTOCOL / OBS_OTLP_ENDPOINT から OTLP MetricExporter を構築する。
func newMetricExporter(ctx context.Context, obsCfg *config.ObservabilityConfig) (sdkmetric.Exporter, error) {
	switch obsCfg.OTLPProtocol() {
	case protocolGRPC:
		var opts []otlpmetricgrpc.Option
		if ep := obsCfg.OTLPEndpoint(); ep != "" {
			opts = append(opts, otlpmetricgrpc.WithEndpointURL(ep))
		}
		return otlpmetricgrpc.New(ctx, opts...)
	case protocolHTTP, "":
		var opts []otlpmetrichttp.Option
		if ep := obsCfg.OTLPEndpoint(); ep != "" {
			opts = append(opts, otlpmetrichttp.WithEndpointURL(ep))
		}
		return otlpmetrichttp.New(ctx, opts...)
	default:
		return nil, errInvalidOTLPProtocol
	}
}

// ProvideTracerProvider は具象 TracerProvider を otel の trace.TracerProvider IF として返す。
func ProvideTracerProvider(tp *sdktrace.TracerProvider) trace.TracerProvider { return tp }

// ProvideMeterProvider は具象 MeterProvider を otel の metric.MeterProvider IF として返す。
// otel 型を DI 層へ漏らさず WorkerMetrics 等へ MeterProvider を注入するための変換点。
func ProvideMeterProvider(mp *sdkmetric.MeterProvider) metric.MeterProvider { return mp }
