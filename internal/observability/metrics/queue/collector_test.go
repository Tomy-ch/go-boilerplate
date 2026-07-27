package queue_test

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	queuemetrics "go-boilerplate/internal/observability/metrics/queue"
	"go-boilerplate/internal/usecase/boundary/worker"
	mock_worker "go-boilerplate/internal/usecase/boundary/worker/mock"
)

// conflictingCollector は、worker_queue_depth と同名だが label 構成が異なる Desc を出す
// テスト用の収集器です。RegisterStatsCollector の衝突エラー分岐を再現するために使います。
type conflictingCollector struct {
	desc *prometheus.Desc
}

func (c conflictingCollector) Describe(ch chan<- *prometheus.Desc) { ch <- c.desc }
func (c conflictingCollector) Collect(_ chan<- prometheus.Metric)  {}

func TestNewStatsCollector(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilなStatsCollectorを構築しCollectorとして登録できる", func(t *testing.T) {
			t.Parallel()

			c := queuemetrics.NewStatsCollector(nil)
			require.NotNil(t, c)
			require.NoError(t, prometheus.NewRegistry().Register(c))
		})
	})
}

func TestStatsCollector_Collect(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("source の visible_not_visible_delayed を depth gauge として出力する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			provider := mock_worker.NewMockQueueStatsProvider(ctrl)
			provider.EXPECT().QueueStats(gomock.Any()).Return(worker.QueueStats{
				Source: worker.QueueDepth{Visible: 10, InFlight: 3, Delayed: 1},
			}, nil)

			c := queuemetrics.NewStatsCollector([]queuemetrics.Target{
				{WorkerName: "email_sender", Adapter: "sqs", Provider: provider},
			})

			expected := `
# HELP worker_queue_depth Approximate number of messages in the queue by state. SQS values are approximate.
# TYPE worker_queue_depth gauge
worker_queue_depth{adapter="sqs",queue="source",state="visible",worker="email_sender"} 10
worker_queue_depth{adapter="sqs",queue="source",state="not_visible",worker="email_sender"} 3
worker_queue_depth{adapter="sqs",queue="source",state="delayed",worker="email_sender"} 1
`
			require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(expected), "worker_queue_depth"))
		})

		t.Run("DLQ があれば queue_dlq の depth も出力する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			provider := mock_worker.NewMockQueueStatsProvider(ctrl)
			dlq := worker.QueueDepth{Visible: 5}
			provider.EXPECT().QueueStats(gomock.Any()).Return(worker.QueueStats{
				Source: worker.QueueDepth{Visible: 1},
				DLQ:    &dlq,
			}, nil)

			c := queuemetrics.NewStatsCollector([]queuemetrics.Target{
				{WorkerName: "w", Adapter: "sqs", Provider: provider},
			})

			expected := `
# HELP worker_queue_depth Approximate number of messages in the queue by state. SQS values are approximate.
# TYPE worker_queue_depth gauge
worker_queue_depth{adapter="sqs",queue="source",state="visible",worker="w"} 1
worker_queue_depth{adapter="sqs",queue="source",state="not_visible",worker="w"} 0
worker_queue_depth{adapter="sqs",queue="source",state="delayed",worker="w"} 0
worker_queue_depth{adapter="sqs",queue="dlq",state="visible",worker="w"} 5
worker_queue_depth{adapter="sqs",queue="dlq",state="not_visible",worker="w"} 0
worker_queue_depth{adapter="sqs",queue="dlq",state="delayed",worker="w"} 0
`
			require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(expected), "worker_queue_depth"))
		})

		t.Run("DLQ が nil なら queue_dlq の depth を出力しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			provider := mock_worker.NewMockQueueStatsProvider(ctrl)
			provider.EXPECT().QueueStats(gomock.Any()).Return(worker.QueueStats{
				Source: worker.QueueDepth{Visible: 1},
			}, nil)

			c := queuemetrics.NewStatsCollector([]queuemetrics.Target{
				{WorkerName: "w", Adapter: "sqs", Provider: provider},
			})

			// source の3系列のみ出力され dlq 系列が無いことを値・ラベルまで検証する。
			expected := `
# HELP worker_queue_depth Approximate number of messages in the queue by state. SQS values are approximate.
# TYPE worker_queue_depth gauge
worker_queue_depth{adapter="sqs",queue="source",state="visible",worker="w"} 1
worker_queue_depth{adapter="sqs",queue="source",state="not_visible",worker="w"} 0
worker_queue_depth{adapter="sqs",queue="source",state="delayed",worker="w"} 0
`
			require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(expected), "worker_queue_depth"))
		})

		t.Run("provider に deadline 付きの context を渡す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			provider := mock_worker.NewMockQueueStatsProvider(ctrl)
			provider.EXPECT().QueueStats(gomock.Any()).DoAndReturn(
				func(ctx context.Context) (worker.QueueStats, error) {
					_, ok := ctx.Deadline()
					assert.True(t, ok)
					return worker.QueueStats{Source: worker.QueueDepth{Visible: 1}}, nil
				})

			c := queuemetrics.NewStatsCollector([]queuemetrics.Target{
				{WorkerName: "w", Adapter: "sqs", Provider: provider},
			})

			assert.Equal(t, 3, testutil.CollectAndCount(c, "worker_queue_depth"))
		})

		t.Run("遅い target が健全な target の収集をブロックしない（並行収集）", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fastCalled := make(chan struct{})
			// 先頭の遅い target は後続の fast が呼ばれるまで待つ。逐次収集だと fast は呼ばれず
			// deadline まで待たされ失敗計上される＝並行でなければ両者成功のアサーションは満たせない。
			slow := mock_worker.NewMockQueueStatsProvider(ctrl)
			slow.EXPECT().QueueStats(gomock.Any()).DoAndReturn(
				func(ctx context.Context) (worker.QueueStats, error) {
					select {
					case <-fastCalled:
						return worker.QueueStats{Source: worker.QueueDepth{Visible: 1}}, nil
					case <-ctx.Done():
						return worker.QueueStats{}, ctx.Err()
					}
				})
			fast := mock_worker.NewMockQueueStatsProvider(ctrl)
			fast.EXPECT().QueueStats(gomock.Any()).DoAndReturn(
				func(context.Context) (worker.QueueStats, error) {
					close(fastCalled)
					return worker.QueueStats{Source: worker.QueueDepth{Visible: 2}}, nil
				})

			c := queuemetrics.NewStatsCollector([]queuemetrics.Target{
				{WorkerName: "slow", Adapter: "sqs", Provider: slow},
				{WorkerName: "fast", Adapter: "sqs", Provider: fast},
			})

			reg := prometheus.NewPedanticRegistry()
			require.NoError(t, reg.Register(c))
			got, err := reg.Gather()
			require.NoError(t, err)

			var depthCount int
			var hasFailure bool
			for _, mf := range got {
				switch mf.GetName() {
				case "worker_queue_depth":
					depthCount = len(mf.GetMetric())
				case "worker_queue_stats_collection_failures_total":
					hasFailure = true
				}
			}
			assert.Equal(t, 6, depthCount) // 2 target × 3 state、両者成功
			assert.False(t, hasFailure)
		})

		t.Run("metric label に queue URL_ARN_message id を含めない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			provider := mock_worker.NewMockQueueStatsProvider(ctrl)
			provider.EXPECT().QueueStats(gomock.Any()).Return(worker.QueueStats{
				Source: worker.QueueDepth{Visible: 1},
			}, nil)

			c := queuemetrics.NewStatsCollector([]queuemetrics.Target{
				{WorkerName: "w", Adapter: "sqs", Provider: provider},
			})

			reg := prometheus.NewPedanticRegistry()
			require.NoError(t, reg.Register(c))
			got, err := reg.Gather()
			require.NoError(t, err)

			require.Len(t, got, 1)
			for _, m := range got[0].GetMetric() {
				for _, l := range m.GetLabel() {
					name := l.GetName()
					assert.NotContains(t, []string{"queue_url", "queue_arn", "message_id", "receipt_handle"}, name)
				}
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("provider エラー時は depth を出さず収集失敗 counter を増やす", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			provider := mock_worker.NewMockQueueStatsProvider(ctrl)
			provider.EXPECT().QueueStats(gomock.Any()).Return(worker.QueueStats{}, assert.AnError)

			c := queuemetrics.NewStatsCollector([]queuemetrics.Target{
				{WorkerName: "w", Adapter: "sqs", Provider: provider},
			})

			// 単一回の収集で、depth は出力されず収集失敗 counter のみが出力されることを確認する
			// （metric 名フィルタを付けないため、出力系列の全体が expected と一致する必要がある）。
			expected := `
# HELP worker_queue_stats_collection_failures_total Total number of queue stats collection failures.
# TYPE worker_queue_stats_collection_failures_total counter
worker_queue_stats_collection_failures_total{adapter="sqs",queue="unknown",worker="w"} 1
`
			require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(expected)))
		})

		t.Run("複数 target 混在で 1 つの失敗が他 target の収集を妨げない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			okProvider := mock_worker.NewMockQueueStatsProvider(ctrl)
			okProvider.EXPECT().QueueStats(gomock.Any()).Return(worker.QueueStats{
				Source: worker.QueueDepth{Visible: 7},
			}, nil)
			ngProvider := mock_worker.NewMockQueueStatsProvider(ctrl)
			ngProvider.EXPECT().QueueStats(gomock.Any()).Return(worker.QueueStats{}, assert.AnError)

			// target A(成功) の後に target B(失敗) を置き、B の失敗が A の depth 出力を妨げないことを確認する。
			c := queuemetrics.NewStatsCollector([]queuemetrics.Target{
				{WorkerName: "a", Adapter: "sqs", Provider: okProvider},
				{WorkerName: "b", Adapter: "sqs", Provider: ngProvider},
			})

			expected := `
# HELP worker_queue_depth Approximate number of messages in the queue by state. SQS values are approximate.
# TYPE worker_queue_depth gauge
worker_queue_depth{adapter="sqs",queue="source",state="visible",worker="a"} 7
worker_queue_depth{adapter="sqs",queue="source",state="not_visible",worker="a"} 0
worker_queue_depth{adapter="sqs",queue="source",state="delayed",worker="a"} 0
# HELP worker_queue_stats_collection_failures_total Total number of queue stats collection failures.
# TYPE worker_queue_stats_collection_failures_total counter
worker_queue_stats_collection_failures_total{adapter="sqs",queue="unknown",worker="b"} 1
`
			require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(expected),
				"worker_queue_depth", "worker_queue_stats_collection_failures_total"))
		})

		t.Run("収集失敗 counter は複数回の Collect で累積する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			provider := mock_worker.NewMockQueueStatsProvider(ctrl)
			provider.EXPECT().QueueStats(gomock.Any()).Return(worker.QueueStats{}, assert.AnError).Times(2)

			c := queuemetrics.NewStatsCollector([]queuemetrics.Target{
				{WorkerName: "w", Adapter: "sqs", Provider: provider},
			})

			// 1 回目の Collect 後は失敗回数が 1。
			first := `
# HELP worker_queue_stats_collection_failures_total Total number of queue stats collection failures.
# TYPE worker_queue_stats_collection_failures_total counter
worker_queue_stats_collection_failures_total{adapter="sqs",queue="unknown",worker="w"} 1
`
			require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(first)))

			// 2 回目の Collect 後は累積して 2 になる。
			second := `
# HELP worker_queue_stats_collection_failures_total Total number of queue stats collection failures.
# TYPE worker_queue_stats_collection_failures_total counter
worker_queue_stats_collection_failures_total{adapter="sqs",queue="unknown",worker="w"} 2
`
			require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(second)))
		})
	})
}

func TestRegisterStatsCollector(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("収集器を登録できる", func(t *testing.T) {
			t.Parallel()

			c := queuemetrics.NewStatsCollector(nil)
			reg := prometheus.NewRegistry()

			require.NoError(t, queuemetrics.RegisterStatsCollector(reg, c))
		})

		t.Run("二重登録はエラーにせず無視する", func(t *testing.T) {
			t.Parallel()

			c := queuemetrics.NewStatsCollector(nil)
			reg := prometheus.NewRegistry()

			require.NoError(t, queuemetrics.RegisterStatsCollector(reg, c))
			require.NoError(t, queuemetrics.RegisterStatsCollector(reg, c))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同名で label 構成が異なるメトリクスが登録済みならエラーを返す", func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			// worker_queue_depth と同名だが label 次元が異なる収集器を先に登録し、衝突させる。
			require.NoError(t, reg.Register(conflictingCollector{
				desc: prometheus.NewDesc("worker_queue_depth", "conflict", []string{"different"}, nil),
			}))

			err := queuemetrics.RegisterStatsCollector(reg, queuemetrics.NewStatsCollector(nil))
			require.Error(t, err)
		})
	})
}
