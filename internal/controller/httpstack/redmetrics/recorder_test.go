package redmetrics

import (
	"net/http"
	"testing"
	"time"

	"go-boilerplate/pkg/xerrors"

	"github.com/prometheus/client_golang/prometheus"
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
	assert.Len(t, descs, 2)
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
