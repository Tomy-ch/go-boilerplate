package instrumentation

import (
	"testing"

	"go-boilerplate/internal/controller/httpstack/redmetrics"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/testkit"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestHTTPRedMetricsMiddleware(t *testing.T) {
	t.Parallel()

	out := HTTPRedMetricsMiddleware(redmetrics.NewPrometheusRecorder())
	assert.Equal(t, httpREDMetricsPriority, out.Middleware.Priority)
	assert.Equal(t, "httpredmetrics", out.Middleware.Name)
	require.NotNil(t, out.Middleware.Middleware)
}

func TestNewHTTPRedMetricsRecorder(t *testing.T) {
	t.Parallel()

	rec := newHTTPRedMetricsRecorder(redmetrics.NewPrometheusRecorder())
	require.NotNil(t, rec)
}

func TestHTTPRedMetricsModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		HTTPRedMetricsModule(),
		fx.Provide(func() prometheus.Registerer { return prometheus.NewRegistry() }),
	)
}
