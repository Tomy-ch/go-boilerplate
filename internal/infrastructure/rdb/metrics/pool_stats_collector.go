// Package metrics は、RDBのコネクションプールの統計情報を収集するための機能を提供します。
package metrics

import (
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "pgxpool"

// poolStatsMetric は、1 メトリクスの説明・型・統計値の取り出し方をまとめた定義です。
type poolStatsMetric struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
	value     func(stats *pgxpool.Stat) float64
}

type PoolStatsCollector struct {
	db      driver.DatabaseDriver
	metrics []poolStatsMetric
}

// New は、PoolStatsCollectorを初期化して返します。
func New(db driver.DatabaseDriver) *PoolStatsCollector {
	gauge := func(name, help string, value func(*pgxpool.Stat) float64) poolStatsMetric {
		return poolStatsMetric{
			desc:      prometheus.NewDesc(prometheus.BuildFQName(namespace, "", name), help, nil, nil),
			valueType: prometheus.GaugeValue,
			value:     value,
		}
	}
	counter := func(name, help string, value func(*pgxpool.Stat) float64) poolStatsMetric {
		return poolStatsMetric{
			desc:      prometheus.NewDesc(prometheus.BuildFQName(namespace, "", name), help, nil, nil),
			valueType: prometheus.CounterValue,
			value:     value,
		}
	}

	return &PoolStatsCollector{
		db: db,
		metrics: []poolStatsMetric{
			gauge("acquired_conns", "Number of currently acquired connections in the pool.",
				func(s *pgxpool.Stat) float64 { return float64(s.AcquiredConns()) }),
			gauge("idle_conns", "Number of currently idle connections in the pool.",
				func(s *pgxpool.Stat) float64 { return float64(s.IdleConns()) }),
			gauge("total_conns", "Total number of connections currently in the pool.",
				func(s *pgxpool.Stat) float64 { return float64(s.TotalConns()) }),
			gauge("constructing_conns", "Number of connections currently being constructed.",
				func(s *pgxpool.Stat) float64 { return float64(s.ConstructingConns()) }),
			gauge("max_conns", "Maximum number of connections allowed in the pool.",
				func(s *pgxpool.Stat) float64 { return float64(s.MaxConns()) }),

			counter("acquire_count_total", "Total number of successful connection acquires from the pool.",
				func(s *pgxpool.Stat) float64 { return float64(s.AcquireCount()) }),
			counter("acquire_duration_seconds_total", "Total duration of all successful connection acquires.",
				func(s *pgxpool.Stat) float64 { return s.AcquireDuration().Seconds() }),
			counter("canceled_acquire_count_total", "Total number of canceled connection acquires.",
				func(s *pgxpool.Stat) float64 { return float64(s.CanceledAcquireCount()) }),
			counter("empty_acquire_count_total", "Total number of acquires that had to create a new connection because the pool was empty.",
				func(s *pgxpool.Stat) float64 { return float64(s.EmptyAcquireCount()) }),
			counter("new_conns_count_total", "Total number of new connections created.",
				func(s *pgxpool.Stat) float64 { return float64(s.NewConnsCount()) }),
			counter("max_lifetime_destroy_count_total", "Total number of connections destroyed due to max lifetime.",
				func(s *pgxpool.Stat) float64 { return float64(s.MaxLifetimeDestroyCount()) }),
			counter("max_idle_destroy_count_total", "Total number of connections destroyed due to max idle time.",
				func(s *pgxpool.Stat) float64 { return float64(s.MaxIdleDestroyCount()) }),
			counter(
				"empty_acquire_wait_time_seconds_total",
				"Total duration of all acquires that had to wait for a connection because the pool was empty.",
				func(s *pgxpool.Stat) float64 { return s.EmptyAcquireWaitTime().Seconds() },
			),
		},
	}
}

// Describe は、PoolStatsCollectorのメトリクスの説明をPrometheusに提供します。
func (c *PoolStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range c.metrics {
		ch <- m.desc
	}
}

// Collect は、PoolStatsCollectorのメトリクスを収集してPrometheusに提供します。
func (c *PoolStatsCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()
	for _, m := range c.metrics {
		ch <- prometheus.MustNewConstMetric(m.desc, m.valueType, m.value(stats))
	}
}

// NewRegisterer は、既定の Prometheus レジストリを Registerer として返します。
func NewRegisterer() prometheus.Registerer {
	return prometheus.DefaultRegisterer
}

// RegisterPoolStatsCollector は、PoolStatsCollectorを指定レジストリに登録します。
// 既に登録済みの場合はエラーを返さず無視します。
func RegisterPoolStatsCollector(reg prometheus.Registerer, c *PoolStatsCollector) error {
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
