package redmetrics

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"go-boilerplate/pkg/xerrors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func collectMetrics(t *testing.T, c prometheus.Collector) []prometheus.Metric {
	t.Helper()

	ch := make(chan prometheus.Metric, 20)
	c.Collect(ch)
	close(ch)

	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}
	return metrics
}

func TestNewPrometheusRecorder(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilなPrometheusRecorderを構築しCollectorとして登録できる", func(t *testing.T) {
			t.Parallel()

			rec := NewPrometheusRecorder()
			require.NotNil(t, rec)
			// prometheus.Collector を満たし registry へ登録できる。
			require.NoError(t, prometheus.NewRegistry().Register(rec))
		})
	})
}

func TestPrometheusRecorder_Describe(t *testing.T) {
	t.Parallel()

	rec := NewPrometheusRecorder()

	ch := make(chan *prometheus.Desc, 20)
	rec.Describe(ch)
	close(ch)

	var descs []*prometheus.Desc
	for d := range ch {
		descs = append(descs, d)
	}

	// requests(Counter) と duration(Histogram) の 2 つの Desc が生成されることを確認します。
	require.Len(t, descs, 2)
	assert.Contains(t, descs[0].String(), "http_server_requests_total")
	assert.Contains(t, descs[1].String(), "http_server_request_duration_seconds")
}

func TestPrometheusRecorder_Observe(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Observe前はメトリクスが生成されない", func(t *testing.T) {
			t.Parallel()

			rec := NewPrometheusRecorder()
			assert.Empty(t, collectMetrics(t, rec))
		})

		t.Run("Observe後はrequestとdurationの2件が生成される", func(t *testing.T) {
			t.Parallel()

			rec := NewPrometheusRecorder()
			rec.Observe(http.MethodGet, "/users/:id", 200, "2xx", 5*time.Millisecond)

			// 同一 label の counter 1 件と histogram 1 件が生成されることを確認します。
			assert.Len(t, collectMetrics(t, rec), 2)
		})

		t.Run("counterの値が1でmethod_route_status_code_status_classラベルが付与される", func(t *testing.T) {
			t.Parallel()

			rec := NewPrometheusRecorder()
			rec.Observe(http.MethodGet, "/users/:id", 200, "2xx", 5*time.Millisecond)

			const expected = `
# HELP http_server_requests_total Total number of HTTP server requests.
# TYPE http_server_requests_total counter
http_server_requests_total{method="GET",route="/users/:id",status_class="2xx",status_code="200"} 1
`
			require.NoError(t, testutil.CollectAndCompare(rec.requests, strings.NewReader(expected), "http_server_requests_total"))
		})

		t.Run("histogramのsample_countが1でmethod_route_status_code_status_classラベルが付与される", func(t *testing.T) {
			t.Parallel()

			rec := NewPrometheusRecorder()
			rec.Observe(http.MethodGet, "/users/:id", 200, "2xx", 5*time.Millisecond)

			labels := prometheus.Labels{
				"method":       http.MethodGet,
				"route":        "/users/:id",
				"status_code":  "200",
				"status_class": "2xx",
			}

			var m dto.Metric
			metric, ok := rec.duration.With(labels).(prometheus.Metric)
			require.True(t, ok)
			require.NoError(t, metric.Write(&m))

			assert.Equal(t, uint64(1), m.GetHistogram().GetSampleCount())

			got := map[string]string{}
			for _, lp := range m.GetLabel() {
				got[lp.GetName()] = lp.GetValue()
			}
			assert.Equal(t, map[string]string{
				"method":       http.MethodGet,
				"route":        "/users/:id",
				"status_code":  "200",
				"status_class": "2xx",
			}, got)
		})
	})
}

func TestRegisterRecorder(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("初回登録は成功する", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			require.NoError(t, RegisterRecorder(reg, NewPrometheusRecorder()))
		})

		t.Run("重複登録の場合はエラーを返さず無視する", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			require.NoError(t, RegisterRecorder(reg, NewPrometheusRecorder()))
			require.NoError(t, RegisterRecorder(reg, NewPrometheusRecorder()))
		})
	})
}

func TestIgnoreAlreadyRegistered(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilはnilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ignoreAlreadyRegistered(nil))
		})

		t.Run("AlreadyRegisteredErrorはnilに丸める", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ignoreAlreadyRegistered(prometheus.AlreadyRegisteredError{}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("その他のエラーはそのまま返す", func(t *testing.T) {
			t.Parallel()
			want := xerrors.New("boom")
			require.ErrorIs(t, ignoreAlreadyRegistered(want), want)
		})
	})
}
