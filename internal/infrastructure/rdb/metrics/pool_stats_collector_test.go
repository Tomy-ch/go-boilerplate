package metrics

import (
	"testing"

	"go-boilerplate/internal/infrastructure/rdb/testkit"

	"github.com/prometheus/client_golang/prometheus"
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
}

func TestRegisterPoolStatsCollector(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("初回登録", func(t *testing.T) {
			reg := prometheus.NewRegistry()
			orig := prometheus.DefaultRegisterer
			prometheus.DefaultRegisterer = reg
			t.Cleanup(func() {
				prometheus.DefaultRegisterer = orig
			})

			db := testkit.NewTestDB(t)
			collector := New(db)

			err := RegisterPoolStatsCollector(collector)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("重複登録", func(t *testing.T) {
			reg := prometheus.NewRegistry()
			orig := prometheus.DefaultRegisterer
			prometheus.DefaultRegisterer = reg
			t.Cleanup(func() {
				prometheus.DefaultRegisterer = orig
			})

			db := testkit.NewTestDB(t)
			collector := New(db)

			// 1回目登録
			err := RegisterPoolStatsCollector(collector)
			require.NoError(t, err)

			// 2回目登録（duplicate）
			err = RegisterPoolStatsCollector(collector)
			require.NoError(t, err)
		})
	})
}
