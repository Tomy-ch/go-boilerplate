package observability

import (
	"context"
	"net/url"

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

// OTLP HTTP の signal 別送出パス。
const (
	otlpTracesPath  = "/v1/traces"
	otlpMetricsPath = "/v1/metrics"
	otlpLogsPath    = "/v1/logs"
)

// errInvalidOTLPProtocol は、OBS_OTLP_PROTOCOL が http/protobuf でも grpc でもない場合に返されます。
var errInvalidOTLPProtocol = xerrors.New("invalid OTLP protocol (want http/protobuf or grpc)")

// ensureOTLPPath は OTLP HTTP のエンドポイント URL に signal パスが無ければ defaultPath を補う。
// otlp*http の WithEndpointURL は path 無し（"" / "/"）の URL に既定 path を補わずルートへ送って 404 に
// なる版があるため、空・ルートのときは defaultPath を明示する。
func ensureOTLPPath(rawURL, defaultPath string) string {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Path != "" && u.Path != "/") {
		return rawURL
	}
	u.Path = defaultPath
	return u.String()
}

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
// TracesEnabled が偽でも span 自体は有効な TraceID / SpanID を持って生成され続け、止まるのは OTLP への
// エクスポートのみ（log-trace 相関はこの経路でも成立する）。sampler は SDK 既定のままで env からは調整できない。
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
	otel.SetTextMapPropagator(NewTextMapPropagator())

	return tp, nil
}

// NewTextMapPropagator は、W3C TraceContext + Baggage の複合 propagator を返します。
func NewTextMapPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// NewMeterProvider は MeterProvider をグローバル登録し、構築した MeterProvider を返す。
// MetricsEnabled が真の場合は Go ランタイムメトリクスの収集 goroutine も開始し、偽の場合は Reader を
// 持たない no-op 相当の MeterProvider を返す。
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
			// mp は既に PeriodicReader（送出 goroutine）を抱えているため、失敗時は Shutdown して回収する。
			_ = mp.Shutdown(context.Background())
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
			opts = append(opts, otlptracehttp.WithEndpointURL(ensureOTLPPath(ep, otlpTracesPath)))
		}
		return otlptracehttp.New(ctx, opts...)
	default:
		return nil, errInvalidOTLPProtocol
	}
}

// newMetricReader は定期収集型の Reader を返す。
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
			opts = append(opts, otlpmetrichttp.WithEndpointURL(ensureOTLPPath(ep, otlpMetricsPath)))
		}
		return otlpmetrichttp.New(ctx, opts...)
	default:
		return nil, errInvalidOTLPProtocol
	}
}

// ProvideTracerProvider は具象 TracerProvider を otel の trace.TracerProvider IF として返す。
func ProvideTracerProvider(tp *sdktrace.TracerProvider) trace.TracerProvider { return tp }

// ProvideMeterProvider は具象 MeterProvider を otel の metric.MeterProvider IF として返す。
func ProvideMeterProvider(mp *sdkmetric.MeterProvider) metric.MeterProvider { return mp }
