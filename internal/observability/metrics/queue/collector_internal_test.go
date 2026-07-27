package queue

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// drainMetrics は、emit が channel へ送出したメトリクスを全件読み出します。
// emit は consumer が読むまでブロックし得るため、送出側を別 goroutine に置きます。
func drainMetrics(t *testing.T, emit func(chan<- prometheus.Metric)) []prometheus.Metric {
	t.Helper()
	ch := make(chan prometheus.Metric)
	go func() {
		emit(ch)
		close(ch)
	}()
	var got []prometheus.Metric
	for m := range ch {
		got = append(got, m)
	}
	return got
}

func Test_StatsCollector_emitFailures(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗が無い場合は何も出力しない", func(t *testing.T) {
			t.Parallel()

			c := NewStatsCollector(nil)

			assert.Empty(t, drainMetrics(t, c.emitFailures))
		})

		t.Run("label 組ごとに累積回数を counter として出力する", func(t *testing.T) {
			t.Parallel()

			c := NewStatsCollector(nil)
			c.recordFailure("w", "sqs", queueUnknown)
			c.recordFailure("w", "sqs", queueUnknown)
			c.recordFailure("w", "sqs", queueDLQ)

			got := drainMetrics(t, c.emitFailures)

			require.Len(t, got, 2) // label 組の数だけ出力される
			byQueue := map[string]float64{}
			for _, m := range got {
				var pb dto.Metric
				require.NoError(t, m.Write(&pb))
				require.NotNil(t, pb.GetCounter())
				for _, l := range pb.GetLabel() {
					if l.GetName() == "queue" {
						byQueue[l.GetValue()] = pb.GetCounter().GetValue()
					}
				}
			}
			assert.InDelta(t, float64(2), byQueue[queueUnknown], 1e-9)
			assert.InDelta(t, float64(1), byQueue[queueDLQ], 1e-9)
		})
	})
}
