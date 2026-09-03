package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go-boilerplate/internal/observability"
)

// failingUpDownMeterProvider は、Int64UpDownCounter 生成に失敗する Meter を返す MeterProvider スタブです。
// NewRealtimeMetrics は最初に Int64UpDownCounter(connections.active) を呼ぶため、生成失敗経路を通せます。
type failingUpDownMeterProvider struct{ embedded.MeterProvider }

// failingUpDownMeter は、Int64UpDownCounter 生成のみエラーを返す Meter スタブです。
type failingUpDownMeter struct{ metric.Meter }

func (failingUpDownMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return failingUpDownMeter{}
}

func (failingUpDownMeter) Int64UpDownCounter(
	string, ...metric.Int64UpDownCounterOption,
) (metric.Int64UpDownCounter, error) {
	return nil, errMeter
}

// newTestRealtimeMetrics は、収集結果を読み出せる RealtimeMetrics を組み立てます。
func newTestRealtimeMetrics(t *testing.T) (*observability.RealtimeMetrics, *sdkmetric.ManualReader) {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	rm, err := observability.NewRealtimeMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	require.NoError(t, err)

	return rm, reader
}

// collectRealtime は、記録済みの metric を名前で引ける形にして返します。
func collectRealtime(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	return rm
}

// sumInt64 は、指定名の Sum[int64] から属性が一致する 1 点の値を取り出します。
func sumInt64(t *testing.T, rm metricdata.ResourceMetrics, name string, attrs ...attribute.KeyValue) int64 {
	t.Helper()

	s, ok := metricByName(t, rm, name).Data.(metricdata.Sum[int64])
	require.True(t, ok, "%s が Sum[int64] であること", name)

	want := attribute.NewSet(attrs...)
	for _, dp := range s.DataPoints {
		if dp.Attributes.Equals(&want) {
			return dp.Value
		}
	}
	t.Fatalf("metric %s に期待した属性の点が無い: %v", name, attrs)

	return 0
}

// histogramCountInt64 は、個数の histogram に記録された点の数を返します。
func histogramCountInt64(t *testing.T, rm metricdata.ResourceMetrics, name string) uint64 {
	t.Helper()

	h, ok := metricByName(t, rm, name).Data.(metricdata.Histogram[int64])
	require.True(t, ok, "%s が Histogram[int64] であること", name)
	require.Len(t, h.DataPoints, 1)

	return h.DataPoints[0].Count
}

// histogramCountFloat64 は、時間の histogram に記録された点の数と合計を返します。
func histogramCountFloat64(t *testing.T, rm metricdata.ResourceMetrics, name string) (uint64, float64) {
	t.Helper()

	h, ok := metricByName(t, rm, name).Data.(metricdata.Histogram[float64])
	require.True(t, ok, "%s が Histogram[float64] であること", name)
	require.Len(t, h.DataPoints, 1)

	return h.DataPoints[0].Count, h.DataPoints[0].Sum
}

func TestNewRealtimeMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Realtime Delivery の全 metric が登録され記録できる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.ConnectionRegistered(ctx)
			rm.ConnectionEstablished(ctx, true)
			rm.ConnectionRejected(ctx, "capacity")
			rm.ConnectionClosed(ctx, "client_gone", 12)
			rm.ReplayExecuted(ctx, "initial", 3)
			rm.ReplayFailed(ctx)
			rm.ReplayStarted(ctx)
			rm.ReplayFinished(ctx)
			rm.ReplayAdmissionTimedOut(ctx)
			rm.CatchUpLag(ctx, 5)
			rm.DeliveryLatency(ctx, 7)
			rm.EventLogAppended(ctx, "ok")
			rm.EventLogLag(ctx, 9)
			rm.WakeupPublishFailed(ctx)
			rm.RecoveryExecuted(ctx, "ok")
			rm.LeaseHeartbeatFailed(ctx)
			rm.CleanupExecuted(ctx, "ok")
			rm.CleanupInstances(ctx, "reclaimed", 2)

			names := map[string]bool{}
			for _, sm := range collectRealtime(t, reader).ScopeMetrics {
				for _, m := range sm.Metrics {
					names[m.Name] = true
				}
			}

			for _, want := range []string{
				"realtime.connections.active", "realtime.connections.accepted",
				"realtime.connections.reconnects", "realtime.connections.rejected",
				"realtime.connections.closed", "realtime.connections.duration_ms",
				"realtime.replay.executions", "realtime.replay.events", "realtime.replay.depth",
				"realtime.replay.failures", "realtime.replay.in_flight", "realtime.replay.admission_timeouts",
				"realtime.catchup.lag_ms", "realtime.delivery.latency_ms",
				"realtime.eventlog.appends", "realtime.eventlog.lag_ms",
				"realtime.wakeup.publish_failures", "realtime.recovery.executions",
				"realtime.lease.heartbeat_failures", "realtime.cleanup.executions", "realtime.cleanup.instances",
			} {
				assert.Contains(t, names, want)
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("計装生成に失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			rm, err := observability.NewRealtimeMetrics(failingUpDownMeterProvider{})

			require.ErrorIs(t, err, errMeter)
			assert.Nil(t, rm)
		})
	})
}

func TestRealtimeMetrics_ConnectionRegistered(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("索引に居る接続数を 1 増やす", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.ConnectionRegistered(ctx)

			got := collectRealtime(t, reader)
			assert.Equal(t, int64(1), sumInt64(t, got, "realtime.connections.active"))
		})

		t.Run("確定を伴わないので受け入れ数は動かさない", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			// 索引に載った直後に 503 で断つ経路があるため、この 2 つは別々に数える。
			rm.ConnectionRegistered(ctx)

			for _, sm := range collectRealtime(t, reader).ScopeMetrics {
				for _, m := range sm.Metrics {
					assert.NotEqual(t, "realtime.connections.accepted", m.Name)
				}
			}
		})
	})
}

func TestRealtimeMetrics_ConnectionEstablished(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("確定した接続数を計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.ConnectionEstablished(ctx, false)

			assert.Equal(t, int64(1), sumInt64(t, collectRealtime(t, reader), "realtime.connections.accepted"))
		})

		t.Run("cursor を伴う接続だけ張り直しとして数える", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.ConnectionEstablished(ctx, true)
			rm.ConnectionEstablished(ctx, false)

			got := collectRealtime(t, reader)
			assert.Equal(t, int64(2), sumInt64(t, got, "realtime.connections.accepted"))
			assert.Equal(t, int64(1), sumInt64(t, got, "realtime.connections.reconnects"))
		})
	})
}

func TestRealtimeMetrics_ConnectionRejected(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("拒否を理由別に計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.ConnectionRejected(ctx, "capacity")
			rm.ConnectionRejected(ctx, "capacity")
			rm.ConnectionRejected(ctx, "degraded")

			got := collectRealtime(t, reader)
			assert.Equal(t, int64(2),
				sumInt64(t, got, "realtime.connections.rejected", attribute.String("reason", "capacity")))
			assert.Equal(t, int64(1),
				sumInt64(t, got, "realtime.connections.rejected", attribute.String("reason", "degraded")))
		})
	})
}

func TestRealtimeMetrics_ConnectionClosed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接続を 1 本減らし理由別に計上して継続時間を記録する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.ConnectionRegistered(ctx)
			rm.ConnectionClosed(ctx, "slow_client", 250)

			got := collectRealtime(t, reader)
			assert.Equal(t, int64(0), sumInt64(t, got, "realtime.connections.active"),
				"載せた分を外したら索引の数はゼロに戻ること")
			assert.Equal(t, int64(1),
				sumInt64(t, got, "realtime.connections.closed", attribute.String("reason", "slow_client")))

			count, sum := histogramCountFloat64(t, got, "realtime.connections.duration_ms")
			assert.Equal(t, uint64(1), count)
			assert.InDelta(t, 250, sum, 0.001)
		})
	})
}

func TestRealtimeMetrics_ReplayExecuted(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("契機別に実行回数と event 数を計上し追いついた件数の分布を残す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.ReplayExecuted(ctx, "catchup", 4)

			got := collectRealtime(t, reader)
			trigger := attribute.String("trigger", "catchup")
			assert.Equal(t, int64(1), sumInt64(t, got, "realtime.replay.executions", trigger))
			assert.Equal(t, int64(4), sumInt64(t, got, "realtime.replay.events", trigger))
			assert.Equal(t, uint64(1), histogramCountInt64(t, got, "realtime.replay.depth"))
		})
	})
}

func TestRealtimeMetrics_ReplayFailed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("読み取り失敗を計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.ReplayFailed(ctx)

			assert.Equal(t, int64(1), sumInt64(t, collectRealtime(t, reader), "realtime.replay.failures"))
		})
	})
}

func TestRealtimeMetrics_ReplayStarted(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("走っている本数を 1 増やす", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.ReplayStarted(ctx)

			assert.Equal(t, int64(1), sumInt64(t, collectRealtime(t, reader), "realtime.replay.in_flight"))
		})
	})
}

func TestRealtimeMetrics_ReplayFinished(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("走っている本数を 1 減らす", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.ReplayStarted(ctx)
			rm.ReplayFinished(ctx)

			assert.Equal(t, int64(0), sumInt64(t, collectRealtime(t, reader), "realtime.replay.in_flight"))
		})
	})
}

func TestRealtimeMetrics_ReplayAdmissionTimedOut(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("枠を待ち切れなかった回数を計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.ReplayAdmissionTimedOut(ctx)

			assert.Equal(t, int64(1),
				sumInt64(t, collectRealtime(t, reader), "realtime.replay.admission_timeouts"))
		})
	})
}

func TestRealtimeMetrics_CatchUpLag(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("wakeup から読み終わりまでの時間を記録する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.CatchUpLag(ctx, 33)

			count, sum := histogramCountFloat64(t, collectRealtime(t, reader), "realtime.catchup.lag_ms")
			assert.Equal(t, uint64(1), count)
			assert.InDelta(t, 33, sum, 0.001)
		})
	})
}

func TestRealtimeMetrics_DeliveryLatency(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("発生から SSE 書き込みまでの時間を記録する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.DeliveryLatency(ctx, 120)

			count, sum := histogramCountFloat64(t, collectRealtime(t, reader), "realtime.delivery.latency_ms")
			assert.Equal(t, uint64(1), count)
			assert.InDelta(t, 120, sum, 0.001)
		})
	})
}

func TestRealtimeMetrics_EventLogAppended(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("追記を結果別に計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.EventLogAppended(ctx, "ok")
			rm.EventLogAppended(ctx, "conflict")

			got := collectRealtime(t, reader)
			assert.Equal(t, int64(1),
				sumInt64(t, got, "realtime.eventlog.appends", attribute.String("result", "ok")))
			assert.Equal(t, int64(1),
				sumInt64(t, got, "realtime.eventlog.appends", attribute.String("result", "conflict")))
		})
	})
}

func TestRealtimeMetrics_EventLogLag(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("outbox 行の作成から追記までの時間を記録する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.EventLogLag(ctx, 80)

			count, sum := histogramCountFloat64(t, collectRealtime(t, reader), "realtime.eventlog.lag_ms")
			assert.Equal(t, uint64(1), count)
			assert.InDelta(t, 80, sum, 0.001)
		})
	})
}

func TestRealtimeMetrics_WakeupPublishFailed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("wakeup の publish 失敗を計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.WakeupPublishFailed(ctx)

			assert.Equal(t, int64(1),
				sumInt64(t, collectRealtime(t, reader), "realtime.wakeup.publish_failures"))
		})
	})
}

func TestRealtimeMetrics_RecoveryExecuted(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("受信先の作り直しを結果別に計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.RecoveryExecuted(ctx, "error")

			assert.Equal(t, int64(1), sumInt64(t, collectRealtime(t, reader),
				"realtime.recovery.executions", attribute.String("result", "error")))
		})
	})
}

func TestRealtimeMetrics_LeaseHeartbeatFailed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("lease の heartbeat 失敗を計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.LeaseHeartbeatFailed(ctx)

			assert.Equal(t, int64(1),
				sumInt64(t, collectRealtime(t, reader), "realtime.lease.heartbeat_failures"))
		})
	})
}

func TestRealtimeMetrics_CleanupExecuted(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ジョブの実行を結果別に計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.CleanupExecuted(ctx, "ok")

			assert.Equal(t, int64(1), sumInt64(t, collectRealtime(t, reader),
				"realtime.cleanup.executions", attribute.String("result", "ok")))
		})
	})
}

func TestRealtimeMetrics_CleanupInstances(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("扱った instance 数を処理のされ方別に計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			rm, reader := newTestRealtimeMetrics(t)

			rm.CleanupInstances(ctx, "detected", 5)
			rm.CleanupInstances(ctx, "skipped", 2)

			got := collectRealtime(t, reader)
			assert.Equal(t, int64(5),
				sumInt64(t, got, "realtime.cleanup.instances", attribute.String("outcome", "detected")))
			assert.Equal(t, int64(2),
				sumInt64(t, got, "realtime.cleanup.instances", attribute.String("outcome", "skipped")))
		})
	})
}
