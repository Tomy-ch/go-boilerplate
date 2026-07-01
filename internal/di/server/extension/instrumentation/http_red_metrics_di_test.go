package instrumentation

import (
	"testing"

	"go-boilerplate/internal/controller/httpstack/redmetrics"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/testkit"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
)

func TestHTTPRedMetricsMiddleware(t *testing.T) {
	t.Parallel()

	out := HTTPRedMetricsMiddleware(redmetrics.NewPrometheusRecorder())
	assert.Equal(t, httpREDMetricsPriority, out.Middleware.Priority)
	assert.Equal(t, "httpredmetrics", out.Middleware.Name)
	assert.NotNil(t, out.Middleware.Middleware)
}

func TestNewHTTPRedMetricsRecorder(t *testing.T) {
	t.Parallel()

	pr := redmetrics.NewPrometheusRecorder()
	rec := newHTTPRedMetricsRecorder(pr)
	// 渡した *PrometheusRecorder をそのまま Recorder として返す（別インスタンスを生成しない）。
	assert.Same(t, pr, rec)
}

func TestHTTPRedMetricsModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		HTTPRedMetricsModule(),
		fx.Provide(func() prometheus.Registerer { return prometheus.NewRegistry() }),
	)
}
