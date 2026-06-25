package observability

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

// NewLoggerProvider は LoggerProvider を構築して返す。
func NewLoggerProvider(obsCfg *config.ObservabilityConfig, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	opts := []sdklog.LoggerProviderOption{sdklog.WithResource(res)}

	if obsCfg.LogsEnabled() {
		exporter, err := newLogExporter(context.Background(), obsCfg)
		if err != nil {
			return nil, xerrors.Wrap(err, "failed to build log exporter")
		}
		opts = append(opts, sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)))
	}

	return sdklog.NewLoggerProvider(opts...), nil
}

// NewLogCore は zap ログを OTLP へ橋渡しする otelzap core を返す。log exporter が無効値のときは nil を返す。
func NewLogCore(
	obsCfg *config.ObservabilityConfig, appCfg *config.ApplicationConfig, lp *sdklog.LoggerProvider,
) logging.LogCore {
	if !obsCfg.LogsEnabled() {
		return nil
	}
	return otelzap.NewCore(appCfg.Name(), otelzap.WithLoggerProvider(lp))
}

// newLogExporter は OBS_OTLP_PROTOCOL / OBS_OTLP_ENDPOINT から OTLP LogExporter を構築する。
func newLogExporter(ctx context.Context, obsCfg *config.ObservabilityConfig) (sdklog.Exporter, error) {
	switch obsCfg.OTLPProtocol() {
	case protocolGRPC:
		var opts []otlploggrpc.Option
		if ep := obsCfg.OTLPEndpoint(); ep != "" {
			opts = append(opts, otlploggrpc.WithEndpointURL(ep))
		}
		return otlploggrpc.New(ctx, opts...)
	case protocolHTTP, "":
		var opts []otlploghttp.Option
		if ep := obsCfg.OTLPEndpoint(); ep != "" {
			opts = append(opts, otlploghttp.WithEndpointURL(ensureOTLPPath(ep, otlpLogsPath)))
		}
		return otlploghttp.New(ctx, opts...)
	default:
		return nil, errInvalidOTLPProtocol
	}
}
