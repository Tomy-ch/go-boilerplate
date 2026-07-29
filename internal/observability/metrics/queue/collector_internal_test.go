package queue

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/usecase/boundary/worker"
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

// labeledValues は、メトリクスを label 組（値の連結）→ 値の map に畳み込みます。
func labeledValues(t *testing.T, metrics []prometheus.Metric) map[string]float64 {
	t.Helper()
	got := map[string]float64{}
	for _, m := range metrics {
		var pb dto.Metric
		require.NoError(t, m.Write(&pb))
		var key strings.Builder
		for _, l := range pb.GetLabel() {
			key.WriteString(l.GetName() + "=" + l.GetValue() + ";")
		}
		got[key.String()] = pb.GetGauge().GetValue()
	}
	return got
}

func TestStatsCollector_collectDepth(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("1 queue の滞留量を state 別の gauge 3 系列として出力する", func(t *testing.T) {
			t.Parallel()

			c := NewStatsCollector(nil)
			target := Target{WorkerName: "email_sender", Adapter: "sqs"}
			depth := worker.QueueDepth{Visible: 10, InFlight: 3, Delayed: 1}

			got := drainMetrics(t, func(ch chan<- prometheus.Metric) {
				c.collectDepth(ch, target, queueSource, depth)
			})

			require.Len(t, got, 3)
			// QueueDepth の各フィールドが state ラベルへ正しく対応することを固定し、
			// Visible / InFlight / Delayed の取り違えを検出する。
			assert.Equal(t, map[string]float64{
				"adapter=sqs;queue=source;state=visible;worker=email_sender;":     10,
				"adapter=sqs;queue=source;state=not_visible;worker=email_sender;": 3,
				"adapter=sqs;queue=source;state=delayed;worker=email_sender;":     1,
			}, labeledValues(t, got))
		})

		t.Run("queue ラベルは引数で切り替わる", func(t *testing.T) {
			t.Parallel()

			c := NewStatsCollector(nil)
			target := Target{WorkerName: "w", Adapter: "sqs"}

			got := drainMetrics(t, func(ch chan<- prometheus.Metric) {
				c.collectDepth(ch, target, queueDLQ, worker.QueueDepth{Visible: 5})
			})

			require.Len(t, got, 3)
			assert.Equal(t, map[string]float64{
				"adapter=sqs;queue=dlq;state=visible;worker=w;":     5,
				"adapter=sqs;queue=dlq;state=not_visible;worker=w;": 0,
				"adapter=sqs;queue=dlq;state=delayed;worker=w;":     0,
			}, labeledValues(t, got))
		})
	})
}

func TestStatsCollector_recordFailure(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一 label 組の失敗は累積する", func(t *testing.T) {
			t.Parallel()

			c := NewStatsCollector(nil)

			c.recordFailure("w", "sqs", queueUnknown)
			c.recordFailure("w", "sqs", queueUnknown)

			assert.Equal(t, int64(2), c.failures[failureKey{worker: "w", adapter: "sqs", queue: queueUnknown}])
		})

		t.Run("label 組が異なれば別カウンタとして数える", func(t *testing.T) {
			t.Parallel()

			c := NewStatsCollector(nil)

			c.recordFailure("w", "sqs", queueUnknown)
			c.recordFailure("w", "sqs", queueDLQ)
			c.recordFailure("other", "sqs", queueUnknown)

			assert.Len(t, c.failures, 3)
			assert.Equal(t, int64(1), c.failures[failureKey{worker: "w", adapter: "sqs", queue: queueUnknown}])
			assert.Equal(t, int64(1), c.failures[failureKey{worker: "w", adapter: "sqs", queue: queueDLQ}])
			assert.Equal(t, int64(1), c.failures[failureKey{worker: "other", adapter: "sqs", queue: queueUnknown}])
		})
	})
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
