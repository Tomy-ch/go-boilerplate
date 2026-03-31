package metrics

import (
	"testing"

	"boilerplate-go/internal/infrastructure/rdb/testkit"

	"github.com/prometheus/client_golang/prometheus"
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
	require.Len(t, descs, 13)
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
	require.Len(t, metrics, 13)
}

func TestRegisterPoolStatsCollector(t *testing.T) {
	db := testkit.NewTestDB(t)
	collector := New(db)

	RegisterPoolStatsCollector(collector)
}
