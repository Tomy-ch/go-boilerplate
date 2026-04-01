// Package metrics は、RDBのコネクションプールの統計情報を収集するための機能を提供します。
package metrics

import (
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/pkg/xerrors"

	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "pgxpool"

type PoolStatsCollector struct {
	db driver.DatabaseDriver

	acquiredConns     *prometheus.Desc
	idleConns         *prometheus.Desc
	totalConns        *prometheus.Desc
	constructingConns *prometheus.Desc
	maxConns          *prometheus.Desc

	acquireCount            *prometheus.Desc
	acquireDuration         *prometheus.Desc
	canceledAcquireCount    *prometheus.Desc
	emptyAcquireCount       *prometheus.Desc
	newConnsCount           *prometheus.Desc
	maxLifetimeDestroyCount *prometheus.Desc
	maxIdleDestroyCount     *prometheus.Desc
	emptyAcquireWaitTime    *prometheus.Desc
}

// New は、PoolStatsCollectorを初期化して返します。
func New(db driver.DatabaseDriver) *PoolStatsCollector {
	return &PoolStatsCollector{
		db: db,

		// Gauges
		acquiredConns: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "acquired_conns"),
			"Number of currently acquired connections in the pool.",
			nil, nil,
		),
		idleConns: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "idle_conns"),
			"Number of currently idle connections in the pool.",
			nil, nil,
		),
		totalConns: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "total_conns"),
			"Total number of connections currently in the pool.",
			nil, nil,
		),
		constructingConns: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "constructing_conns"),
			"Number of connections currently being constructed.",
			nil, nil,
		),
		maxConns: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "max_conns"),
			"Maximum number of connections allowed in the pool.",
			nil, nil,
		),

		// Counters
		acquireCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "acquire_count_total"),
			"Total number of successful connection acquires from the pool.",
			nil, nil,
		),
		acquireDuration: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "acquire_duration_seconds_total"),
			"Total duration of all successful connection acquires.",
			nil, nil,
		),
		canceledAcquireCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "canceled_acquire_count_total"),
			"Total number of  canceled connection acquires.",
			nil, nil,
		),
		emptyAcquireCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "empty_acquire_count_total"),
			"Total number of acquires that had to create a new connection because the pool was empty.",
			nil, nil,
		),
		newConnsCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "new_conns_count_total"),
			"Total number of new connections created.",
			nil, nil,
		),
		maxLifetimeDestroyCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "max_lifetime_destroy_count_total"),
			"Total number of connections destroyed due to max lifetime.",
			nil, nil,
		),
		maxIdleDestroyCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "max_idle_destroy_count_total"),
			"Total number of connections destroyed due to max idle time.",
			nil, nil,
		),
		emptyAcquireWaitTime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "empty_acquire_wait_time_seconds_total"),
			"Total duration of all acquires that had to wait for a connection because the pool was empty.",
			nil, nil,
		),
	}
}

// Describe は、PoolStatsCollectorのメトリクスの説明をPrometheusに提供します。
func (c *PoolStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquiredConns
	ch <- c.idleConns
	ch <- c.totalConns
	ch <- c.constructingConns
	ch <- c.maxConns

	ch <- c.acquireCount
	ch <- c.acquireDuration
	ch <- c.canceledAcquireCount
	ch <- c.emptyAcquireCount
	ch <- c.newConnsCount
	ch <- c.maxLifetimeDestroyCount
	ch <- c.maxIdleDestroyCount
	ch <- c.emptyAcquireWaitTime
}

// Collect は、PoolStatsCollectorのメトリクスを収集してPrometheusに提供します。
func (c *PoolStatsCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()

	ch <- prometheus.MustNewConstMetric(c.acquiredConns, prometheus.GaugeValue, float64(stats.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(stats.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(stats.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.constructingConns, prometheus.GaugeValue, float64(stats.ConstructingConns()))
	ch <- prometheus.MustNewConstMetric(c.maxConns, prometheus.GaugeValue, float64(stats.MaxConns()))

	ch <- prometheus.MustNewConstMetric(c.acquireCount, prometheus.CounterValue, float64(stats.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.acquireDuration, prometheus.CounterValue, stats.AcquireDuration().Seconds())
	ch <- prometheus.MustNewConstMetric(c.canceledAcquireCount, prometheus.CounterValue, float64(stats.CanceledAcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.emptyAcquireCount, prometheus.CounterValue, float64(stats.EmptyAcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.newConnsCount, prometheus.CounterValue, float64(stats.NewConnsCount()))
	ch <- prometheus.MustNewConstMetric(c.maxLifetimeDestroyCount, prometheus.CounterValue, float64(stats.MaxLifetimeDestroyCount()))
	ch <- prometheus.MustNewConstMetric(c.maxIdleDestroyCount, prometheus.CounterValue, float64(stats.MaxIdleDestroyCount()))
	ch <- prometheus.MustNewConstMetric(c.emptyAcquireWaitTime, prometheus.CounterValue, stats.EmptyAcquireWaitTime().Seconds())
}

// RegisterPoolStatsCollector は、PoolStatsCollectorをPrometheusのレジストリに登録します。
func RegisterPoolStatsCollector(c *PoolStatsCollector) error {
	err := prometheus.Register(c)
	if err != nil {
		var alreadyRegisteredErr prometheus.AlreadyRegisteredError
		if xerrors.As(err, &alreadyRegisteredErr) {
			return nil
		}
		return err
	}
	return nil
}
