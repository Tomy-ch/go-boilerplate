// Package queue は、worker が対象とする broker queue の滞留量（depth / DLQ count）を
// Prometheus メトリクスとして公開する収集器を提供します。
//
// engine は worker.QueueStatsProvider を知りません。収集器が capability を実装する adapter
// （SQS など）からのみ滞留量を pull します。scrape のたびに provider 経由で broker API を呼ぶため、
// SQS では値が approximate（厳密な件数ではなく滞留傾向の gauge）になります。
package queue

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/pkg/xerrors"
)

const (
	namespace = "worker"
	subsystem = "queue"
)

// collectTimeout は、1 回の scrape（Collect）で全 target の滞留量取得に許す上限時間です。
// registry は Gather のたびに Collect を goroutine で並行起動するため、deadline なしで
// broker API（SQS GetQueueAttributes 等）を呼ぶと、応答不能時に goroutine が蓄積して
// リソースを枯渇させ得ます。これを防ぐため scrape 単位で timeout を設けます。
// 将来 TTL cache を導入する際は、その更新間隔に合わせてこの値を見直す想定です。
const collectTimeout = 5 * time.Second

// queue label の値。URL / ARN / message id 等の高カーディナリティ・秘匿情報は label にしません。
const (
	queueSource  = "source"
	queueDLQ     = "dlq"
	queueUnknown = "unknown"
)

// state label の値（QueueDepth のフィールド対応）。
const (
	stateVisible    = "visible"
	stateNotVisible = "not_visible"
	stateDelayed    = "delayed"
)

// Target は、収集対象の worker と滞留量取得 capability の対応です。
// DI group "worker.queue_stats_targets" 経由で SQS worker のときだけ任意登録されます。
type Target struct {
	// WorkerName は、metric の worker label に使う worker 名です。
	WorkerName string
	// Adapter は、metric の adapter label に使う broker adapter 種別（例: "sqs"）です。
	Adapter string
	// Provider は、滞留量を取得する capability 実装です。
	Provider worker.QueueStatsProvider
}

// StatsCollector は、QueueStatsProvider capability を呼んで滞留量を公開する Prometheus 収集器です。
type StatsCollector struct {
	targets []Target

	depthDesc   *prometheus.Desc
	failureDesc *prometheus.Desc

	// failures は、収集失敗の累積回数です。失敗は Collect（scrape）時にのみ発生するため、
	// counter の単調増加を Collect 内 increment で表現します。
	mu       sync.Mutex
	failures map[failureKey]int64
}

// failureKey は、収集失敗 counter の label 組です。
type failureKey struct {
	worker  string
	adapter string
	queue   string
}

// NewStatsCollector は、指定の収集対象から StatsCollector を生成します。
func NewStatsCollector(targets []Target) *StatsCollector {
	return &StatsCollector{
		targets: targets,
		depthDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "depth"),
			"Approximate number of messages in the queue by state. SQS values are approximate.",
			[]string{"worker", "adapter", "queue", "state"}, nil,
		),
		failureDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "stats_collection_failures_total"),
			"Total number of queue stats collection failures.",
			[]string{"worker", "adapter", "queue"}, nil,
		),
		failures: make(map[failureKey]int64),
	}
}

// Describe は、収集器のメトリクス説明を Prometheus に提供します。
func (c *StatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.depthDesc
	ch <- c.failureDesc
}

// Collect は、各 target から滞留量を取得して Prometheus に提供します。
// provider がエラーを返した場合は depth を出さず、収集失敗 counter を増やします。
func (c *StatsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), collectTimeout)
	defer cancel()
	for _, t := range c.targets {
		stats, err := t.Provider.QueueStats(ctx)
		if err != nil {
			// QueueStats は source / DLQ いずれの失敗かを区別しないため queue="unknown" とします。
			c.recordFailure(t.WorkerName, t.Adapter, queueUnknown)
			continue
		}

		c.collectDepth(ch, t, queueSource, stats.Source)
		if stats.DLQ != nil {
			c.collectDepth(ch, t, queueDLQ, *stats.DLQ)
		}
	}
	c.emitFailures(ch)
}

// collectDepth は、1 queue の滞留量を state 別の gauge として出力します。
func (c *StatsCollector) collectDepth(ch chan<- prometheus.Metric, t Target, queue string, d worker.QueueDepth) {
	ch <- prometheus.MustNewConstMetric(
		c.depthDesc, prometheus.GaugeValue, float64(d.Visible), t.WorkerName, t.Adapter, queue, stateVisible)
	ch <- prometheus.MustNewConstMetric(
		c.depthDesc, prometheus.GaugeValue, float64(d.InFlight), t.WorkerName, t.Adapter, queue, stateNotVisible)
	ch <- prometheus.MustNewConstMetric(
		c.depthDesc, prometheus.GaugeValue, float64(d.Delayed), t.WorkerName, t.Adapter, queue, stateDelayed)
}

// recordFailure は、収集失敗の累積回数を増やします。
func (c *StatsCollector) recordFailure(workerName, adapter, queue string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures[failureKey{worker: workerName, adapter: adapter, queue: queue}]++
}

// emitFailures は、累積した収集失敗回数を counter として出力します。
func (c *StatsCollector) emitFailures(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.failures {
		ch <- prometheus.MustNewConstMetric(
			c.failureDesc, prometheus.CounterValue, float64(v), k.worker, k.adapter, k.queue)
	}
}

// RegisterStatsCollector は、StatsCollector を指定レジストリに登録します。
// 既に登録済みの場合はエラーを返さず無視します。
func RegisterStatsCollector(reg prometheus.Registerer, c *StatsCollector) error {
	err := reg.Register(c)
	if err != nil {
		var alreadyRegisteredErr prometheus.AlreadyRegisteredError
		if xerrors.As(err, &alreadyRegisteredErr) {
			return nil
		}
		return err
	}
	return nil
}
