package instrumentation

import (
	"go-boilerplate/internal/controller/httpstack/redmetrics"
	"go-boilerplate/internal/di/server/extension"

	"go.uber.org/fx"
)

// httpRedMetricsPriority は、HTTP RED メトリクスミドルウェアの適用順序です。
// 順序設計の根拠は README「Priority Order」を参照してください。
const httpRedMetricsPriority = 8

// HTTPRedMetricsModule は、HTTP RED（Rate / Errors / Duration）メトリクスのミドルウェアを提供するfxモジュールを返します。
func HTTPRedMetricsModule() fx.Option {
	return fx.Module("mw.httpredmetrics",
		fx.Provide(
			redmetrics.NewPrometheusRecorder,
			newHTTPRedMetricsRecorder,
			HTTPRedMetricsMiddleware,
		),
		fx.Invoke(
			redmetrics.RegisterRecorder,
		),
	)
}

// newHTTPRedMetricsRecorder は、PrometheusRecorder を Recorder インターフェースとして提供します。
func newHTTPRedMetricsRecorder(r *redmetrics.PrometheusRecorder) redmetrics.Recorder {
	return r
}

// HTTPRedMetricsMiddleware は、HTTP RED メトリクスを計測するミドルウェアを提供します。
func HTTPRedMetricsMiddleware(rec redmetrics.Recorder) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "httpredmetrics",
			Priority:   httpRedMetricsPriority,
			Middleware: redmetrics.Middleware(rec),
		},
	}
}
