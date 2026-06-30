package metrics

import (
	"testing"

	"go-boilerplate/internal/infrastructure/rdb/testkit"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	db := testkit.NewTestDB(t)
	collector := New(db)
	require.NotNil(t, collector)
}

func TestPoolStatsCollector_Describe(t *testing.T) {
	t.Parallel()

	db := testkit.NewTestDB(t)
	collector := New(db)

	ch := make(chan *prometheus.Desc, 20)
	collector.Describe(ch)
	close(ch)

	var descs []*prometheus.Desc
	for d := range ch {
		descs = append(descs, d)
	}

	// Gaugeのメトリクスが5つ、Counterのメトリクスが8つ、合計13個のDescが生成されることを確認します。
	assert.Len(t, descs, 13)
}

func TestPoolStatsCollector_Collect(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("合計13個のメトリクスを出力する", func(t *testing.T) {
			t.Parallel()

			db := testkit.NewTestDB(t)
			collector := New(db)

			ch := make(chan prometheus.Metric, 20)
			collector.Collect(ch)
			close(ch)

			var metrics []prometheus.Metric
			for m := range ch {
				metrics = append(metrics, m)
			}

			// Gaugeのメトリクスが5つ、Counterのメトリクスが8つ、合計13個のMetricが生成されることを確認します。
			assert.Len(t, metrics, 13)
		})

		t.Run("max_connsをゲージ値・型付きで出力する", func(t *testing.T) {
			t.Parallel()

			db := testkit.NewTestDB(t)
			collector := New(db)

			reg := prometheus.NewRegistry()
			require.NoError(t, reg.Register(collector))

			families, err := reg.Gather()
			require.NoError(t, err)

			var maxConns *dto.MetricFamily
			for _, mf := range families {
				if mf.GetName() == "pgxpool_max_conns" {
					maxConns = mf
					break
				}
			}
			require.NotNil(t, maxConns)

			// max_conns は Gauge 型で、プールの MaxConns 設定値がそのまま出力される。
			assert.Equal(t, dto.MetricType_GAUGE, maxConns.GetType())
			require.Len(t, maxConns.GetMetric(), 1)
			expected := float64(db.Stats().MaxConns())
			assert.InDelta(t, expected, maxConns.GetMetric()[0].GetGauge().GetValue(), 0)
		})
	})
}

func TestNewRegisterer(t *testing.T) {
	t.Parallel()

	assert.Equal(t, prometheus.DefaultRegisterer, NewRegisterer())
}

func TestRegisterPoolStatsCollector(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("初回登録", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			db := testkit.NewTestDB(t)
			collector := New(db)

			err := RegisterPoolStatsCollector(reg, collector)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("重複登録の場合はエラーを返さず無視する", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			db := testkit.NewTestDB(t)
			collector := New(db)

			err := RegisterPoolStatsCollector(reg, collector)
			require.NoError(t, err)

			err = RegisterPoolStatsCollector(reg, collector)
			require.NoError(t, err)
		})
	})
}
