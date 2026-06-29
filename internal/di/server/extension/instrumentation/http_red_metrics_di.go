package instrumentation

import (
	"go-boilerplate/internal/controller/httpstack/redmetrics"
	"go-boilerplate/internal/di/server/extension"

	"go.uber.org/fx"
)

// httpREDMetricsPriority は、HTTP RED メトリクスミドルウェアの適用順序です。
// logging(8) の後・cookie(10) の前に置き、observability(2) 開始後に計測します。
// forceJSON(7) と重複しないよう 9 を用います（Priority 重複は適用時にエラー）。
const httpREDMetricsPriority = 9

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
			Priority:   httpREDMetricsPriority,
			Middleware: redmetrics.Middleware(rec),
		},
	}
}
