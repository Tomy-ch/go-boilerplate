package observability

import (
	"context"
	"net/url"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

// otlpLogsPath は OTLP HTTP のログ送出パス。
const otlpLogsPath = "/v1/logs"

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
			opts = append(opts, otlploghttp.WithEndpointURL(ensureLogsPath(ep)))
		}
		return otlploghttp.New(ctx, opts...)
	default:
		return nil, errInvalidOTLPProtocol
	}
}

// ensureLogsPath は OTLP HTTP のエンドポイント URL に path が無ければ /v1/logs を補う。
// otlploghttp v0.20.0 の WithEndpointURL は path 無し URL のとき既定 /v1/logs を補わず
// （空文字を「設定済み」と解釈する）ルートへ送って 404 になるための回避。
func ensureLogsPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Path != "" {
		return rawURL
	}
	u.Path = otlpLogsPath
	return u.String()
}
