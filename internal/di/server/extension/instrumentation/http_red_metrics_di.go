package instrumentation

import (
	"go-boilerplate/internal/controller/httpstack/redmetrics"
	"go-boilerplate/internal/di/server/extension"

	"go.uber.org/fx"
)

// httpREDMetricsPriority は、HTTP RED メトリクスミドルウェアの適用順序です。
// forceJSON(7) の後・logging(9) の前に置き、observability(2) 開始後に計測します。
// logging より外側に置くことで、redmetrics の After フックが logging の After より先に発火し、
// 計測した duration に logging の I/O が混入しないようにする。
// forceJSON(7) と重複しないよう 8 を用います（Priority 重複は適用時にエラー）。
const httpREDMetricsPriority = 8

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
