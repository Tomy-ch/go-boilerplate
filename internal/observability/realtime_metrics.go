package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// realtimeMeterName は、Realtime Delivery 計装の meter 名です。
const realtimeMeterName = "go-boilerplate/realtime"

// Realtime Delivery の計装が使う属性キー。ここに無いキーは付けず、主体の識別子は label にしない
// （docs/design/realtime-delivery.md §3.4。internal/architest/realtime_metrics_test.go が検査する）。
const (
	// attrReason は、接続を受け付けなかった・閉じた理由です。
	attrReason = "reason"
	// attrTrigger は、replay を起こした契機です。
	attrTrigger = "trigger"
	// attrResult は、1 回の操作の成否です。
	attrResult = "result"
	// attrOutcome は、掃除の対象 1 件がどう処理されたかです。
	attrOutcome = "outcome"
)

// label に載せる値の語彙。3 つの process が同じ語で記録するよう、計装の側で定義します。
const (
	// RealtimeResultOK は、操作が成功したことを表します。
	RealtimeResultOK = "ok"
	// RealtimeResultError は、操作が失敗したことを表します。
	RealtimeResultError = "error"
	// RealtimeResultConflict は、既に同じ位置が書かれていて追記しなかったことを表します。
	RealtimeResultConflict = "conflict"
)

// RealtimeMetrics は、Realtime Delivery の lifecycle 計装一式です。
// serve（接続・replay・配送）・relay（EventLog への追記）・cleanup job（lease の回収）が同じ meter を共有します。
type RealtimeMetrics struct {
	connectionsActive     metric.Int64UpDownCounter
	connectionsAccepted   metric.Int64Counter
	connectionsReconnects metric.Int64Counter
	connectionsRejected   metric.Int64Counter
	connectionsClosed     metric.Int64Counter
	connectionDurationMs  metric.Float64Histogram

	replayExecutions        metric.Int64Counter
	replayEvents            metric.Int64Counter
	replayDepth             metric.Int64Histogram
	replayFailures          metric.Int64Counter
	replayInFlight          metric.Int64UpDownCounter
	replayAdmissionTimeouts metric.Int64Counter
	catchUpLagMs            metric.Float64Histogram

	deliveryLatencyMs     metric.Float64Histogram
	eventLogAppends       metric.Int64Counter
	eventLogLagMs         metric.Float64Histogram
	wakeupPublishFailures metric.Int64Counter
	recoveryExecutions    metric.Int64Counter

	leaseHeartbeatFailures metric.Int64Counter
	cleanupExecutions      metric.Int64Counter
	cleanupInstances       metric.Int64Counter
}

// NewRealtimeMetrics は、注入された MeterProvider から Realtime Delivery の計装一式を生成します。
// いずれかの計装生成失敗で error を返します。
func NewRealtimeMetrics(mp metric.MeterProvider) (*RealtimeMetrics, error) {
	b := &meterBuilder{m: mp.Meter(realtimeMeterName)}
	rm := &RealtimeMetrics{
		connectionsActive:     b.upDownCounter("realtime.connections.active", "この instance が索引に持っている SSE 接続数"),
		connectionsAccepted:   b.counter("realtime.connections.accepted", "レスポンスを確定して配信を始めた接続数"),
		connectionsReconnects: b.counter("realtime.connections.reconnects", "cursor を伴って張り直された接続数"),
		connectionsRejected:   b.counter("realtime.connections.rejected", "確定せず 503 で断った接続数"),
		connectionsClosed:     b.counter("realtime.connections.closed", "索引から外れた接続数"),
		connectionDurationMs:  b.histogram("realtime.connections.duration_ms", "接続が索引に居た時間(ミリ秒)"),

		replayExecutions:        b.counter("realtime.replay.executions", "EventLog を読み進めた回数"),
		replayEvents:            b.counter("realtime.replay.events", "読み進めて送出した event 数"),
		replayDepth:             b.countHistogram("realtime.replay.depth", "1 回の読み進めで追いついた event 数"),
		replayFailures:          b.counter("realtime.replay.failures", "EventLog の読み取りに失敗した回数"),
		replayInFlight:          b.upDownCounter("realtime.replay.in_flight", "同時に走っている読み進めの本数"),
		replayAdmissionTimeouts: b.counter("realtime.replay.admission_timeouts", "初回 replay の枠を待ち切れず諦めた回数"),
		catchUpLagMs:            b.histogram("realtime.catchup.lag_ms", "wakeup 受信から読み終わりまでの時間(ミリ秒)"),

		deliveryLatencyMs:     b.histogram("realtime.delivery.latency_ms", "event の発生から SSE へ書き終わるまでの時間(ミリ秒)"),
		eventLogAppends:       b.counter("realtime.eventlog.appends", "EventLog への追記回数"),
		eventLogLagMs:         b.histogram("realtime.eventlog.lag_ms", "outbox 行の作成から EventLog 追記までの時間(ミリ秒)"),
		wakeupPublishFailures: b.counter("realtime.wakeup.publish_failures", "wakeup の publish に失敗した回数"),
		recoveryExecutions:    b.counter("realtime.recovery.executions", "消えた受信先を作り直した回数"),

		leaseHeartbeatFailures: b.counter("realtime.lease.heartbeat_failures", "instance lease の heartbeat に失敗した回数"),
		cleanupExecutions:      b.counter("realtime.cleanup.executions", "orphan cleanup ジョブの実行回数"),
		cleanupInstances:       b.counter("realtime.cleanup.instances", "orphan cleanup が扱った instance 数"),
	}
	if b.err != nil {
		return nil, b.err
	}
	return rm, nil
}

// ConnectionRegistered は、接続が索引に載ったことを計上します。
// 確定はまだで、この後 503 で断ることがあります（対になるのは ConnectionClosed）。
func (m *RealtimeMetrics) ConnectionRegistered(ctx context.Context) {
	m.connectionsActive.Add(ctx, 1)
}

// ConnectionEstablished は、レスポンスを確定して配信を始めたことを計上します。
// resumed は cursor を伴う張り直しです。
func (m *RealtimeMetrics) ConnectionEstablished(ctx context.Context, resumed bool) {
	m.connectionsAccepted.Add(ctx, 1)
	if resumed {
		m.connectionsReconnects.Add(ctx, 1)
	}
}

// ConnectionRejected は、確定せず 503 で断ったことを理由別に計上します。
func (m *RealtimeMetrics) ConnectionRejected(ctx context.Context, reason string) {
	m.connectionsRejected.Add(ctx, 1, metric.WithAttributes(attribute.String(attrReason, reason)))
}

// ConnectionClosed は、索引から外れた接続を理由別に計上し、索引に居た時間を記録します。
func (m *RealtimeMetrics) ConnectionClosed(ctx context.Context, reason string, durationMs float64) {
	m.connectionsActive.Add(ctx, -1)
	m.connectionsClosed.Add(ctx, 1, metric.WithAttributes(attribute.String(attrReason, reason)))
	m.connectionDurationMs.Record(ctx, durationMs)
}

// ReplayExecuted は、1 回の読み進めを契機別に計上し、送出できた event 数を記録します。
func (m *RealtimeMetrics) ReplayExecuted(ctx context.Context, trigger string, events int64) {
	attrs := metric.WithAttributes(attribute.String(attrTrigger, trigger))
	m.replayExecutions.Add(ctx, 1, attrs)
	m.replayEvents.Add(ctx, events, attrs)
	m.replayDepth.Record(ctx, events, attrs)
}

// ReplayFailed は、EventLog の読み取り失敗を計上します。
func (m *RealtimeMetrics) ReplayFailed(ctx context.Context) { m.replayFailures.Add(ctx, 1) }

// ReplayStarted は、読み進めが 1 本始まったことを記録します。
func (m *RealtimeMetrics) ReplayStarted(ctx context.Context) { m.replayInFlight.Add(ctx, 1) }

// ReplayFinished は、読み進めが 1 本終わったことを記録します。
func (m *RealtimeMetrics) ReplayFinished(ctx context.Context) { m.replayInFlight.Add(ctx, -1) }

// ReplayAdmissionTimedOut は、初回 replay の枠を待ち切れなかったことを計上します。
func (m *RealtimeMetrics) ReplayAdmissionTimedOut(ctx context.Context) {
	m.replayAdmissionTimeouts.Add(ctx, 1)
}

// CatchUpLag は、wakeup を受け取ってから読み終わるまでの時間を記録します。
func (m *RealtimeMetrics) CatchUpLag(ctx context.Context, lagMs float64) {
	m.catchUpLagMs.Record(ctx, lagMs)
}

// DeliveryLatency は、event の発生から SSE へ書き終わるまでの時間を記録します。
// 2 つの instance をまたぐ差分なので、時計のずれを含む近似です。
func (m *RealtimeMetrics) DeliveryLatency(ctx context.Context, latencyMs float64) {
	m.deliveryLatencyMs.Record(ctx, latencyMs)
}

// EventLogAppended は、EventLog への追記を結果別に計上します。
func (m *RealtimeMetrics) EventLogAppended(ctx context.Context, result string) {
	m.eventLogAppends.Add(ctx, 1, metric.WithAttributes(attribute.String(attrResult, result)))
}

// EventLogLag は、outbox 行の作成から EventLog 追記までの時間を記録します。
func (m *RealtimeMetrics) EventLogLag(ctx context.Context, lagMs float64) {
	m.eventLogLagMs.Record(ctx, lagMs)
}

// WakeupPublishFailed は、wakeup の publish 失敗を計上します。
func (m *RealtimeMetrics) WakeupPublishFailed(ctx context.Context) {
	m.wakeupPublishFailures.Add(ctx, 1)
}

// RecoveryExecuted は、消えた受信先の作り直しを結果別に計上します。
func (m *RealtimeMetrics) RecoveryExecuted(ctx context.Context, result string) {
	m.recoveryExecutions.Add(ctx, 1, metric.WithAttributes(attribute.String(attrResult, result)))
}

// LeaseHeartbeatFailed は、instance lease の heartbeat 失敗を計上します。
func (m *RealtimeMetrics) LeaseHeartbeatFailed(ctx context.Context) {
	m.leaseHeartbeatFailures.Add(ctx, 1)
}

// CleanupExecuted は、orphan cleanup ジョブ 1 回の実行を結果別に計上します。
func (m *RealtimeMetrics) CleanupExecuted(ctx context.Context, result string) {
	m.cleanupExecutions.Add(ctx, 1, metric.WithAttributes(attribute.String(attrResult, result)))
}

// CleanupInstances は、orphan cleanup が扱った instance 数を処理のされ方別に計上します。
func (m *RealtimeMetrics) CleanupInstances(ctx context.Context, outcome string, n int64) {
	m.cleanupInstances.Add(ctx, n, metric.WithAttributes(attribute.String(attrOutcome, outcome)))
}
