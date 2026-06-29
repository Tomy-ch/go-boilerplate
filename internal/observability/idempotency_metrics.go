package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// idempotencyMeterName は、冪等性サブシステム計装の meter 名です。
const idempotencyMeterName = "go-boilerplate/idempotency"

// idempotencyGCJob は、GC cleanup カウンタの job ラベル固定値です。
const idempotencyGCJob = "idempotency_gc"

// IdempotencyMetrics は、冪等性サブシステムの判定結果・内部失敗・GC 削除を計上する計装一式です。
// usecase 層の idempotency.Metrics および idempotency.GCMetrics を実装します。
//
// 高カーディナリティ・秘匿値（Idempotency-Key / scope / fingerprint / PII / raw error）は
// ラベルに載せません。許可ラベルは operation_id / result / phase / job のみです。
type IdempotencyMetrics struct {
	requests       metric.Int64Counter
	failures       metric.Int64Counter
	expiredCleanup metric.Int64Counter
}

// NewIdempotencyMetrics は、注入された MeterProvider から冪等性計装一式を生成します。
// いずれかの計装生成失敗で error を返します。
func NewIdempotencyMetrics(mp metric.MeterProvider) (*IdempotencyMetrics, error) {
	b := &meterBuilder{m: mp.Meter(idempotencyMeterName)}
	im := &IdempotencyMetrics{
		requests:       b.counter("idempotency.requests", "冪等性判定結果数(result=hit/miss/conflict/fingerprint_mismatch 別)"),
		failures:       b.counter("idempotency.failures", "冪等性の内部失敗数(phase=claim/complete/gc_cleanup 別)"),
		expiredCleanup: b.counter("idempotency.expired_cleanup", "GC が削除した失効キー件数"),
	}
	if b.err != nil {
		return nil, b.err
	}
	return im, nil
}

// IncHit は、completed 行の再送（replay）を計上します。
func (m *IdempotencyMetrics) IncHit(ctx context.Context, operationID string) {
	m.incRequest(ctx, operationID, "hit")
}

// IncMiss は、新規 claim 成立を計上します。
func (m *IdempotencyMetrics) IncMiss(ctx context.Context, operationID string) {
	m.incRequest(ctx, operationID, "miss")
}

// IncConflict は、処理中キーへの並行再送（409）を計上します。
func (m *IdempotencyMetrics) IncConflict(ctx context.Context, operationID string) {
	m.incRequest(ctx, operationID, "conflict")
}

// IncFingerprintMismatch は、同一キー別ボディの再利用（422）を計上します。
func (m *IdempotencyMetrics) IncFingerprintMismatch(ctx context.Context, operationID string) {
	m.incRequest(ctx, operationID, "fingerprint_mismatch")
}

// IncClaimFailure は、ErrLockTimeout 以外の Claim 失敗を計上します。
func (m *IdempotencyMetrics) IncClaimFailure(ctx context.Context, operationID string) {
	m.incFailure(ctx, operationID, "claim")
}

// IncCompleteFailure は、Complete 失敗（結果保存失敗）を計上します。
func (m *IdempotencyMetrics) IncCompleteFailure(ctx context.Context, operationID string) {
	m.incFailure(ctx, operationID, "complete")
}

// IncExpiredCleanup は、削除に成功した失効キー件数を計上します。
func (m *IdempotencyMetrics) IncExpiredCleanup(ctx context.Context, count int64) {
	m.expiredCleanup.Add(ctx, count,
		metric.WithAttributes(attribute.String("job", idempotencyGCJob)))
}

// IncExpiredCleanupFailure は、GC 削除バッチの失敗回数を計上します。
func (m *IdempotencyMetrics) IncExpiredCleanupFailure(ctx context.Context) {
	m.incFailure(ctx, "", "gc_cleanup")
}

func (m *IdempotencyMetrics) incRequest(ctx context.Context, operationID, result string) {
	m.requests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation_id", normalizeOperationID(operationID)),
		attribute.String("result", result),
	))
}

func (m *IdempotencyMetrics) incFailure(ctx context.Context, operationID, phase string) {
	m.failures.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation_id", normalizeOperationID(operationID)),
		attribute.String("phase", phase),
	))
}

// normalizeOperationID は、空の operationID を unknown へ丸めます（空ラベル回避）。
func normalizeOperationID(operationID string) string {
	if operationID == "" {
		return "unknown"
	}
	return operationID
}
