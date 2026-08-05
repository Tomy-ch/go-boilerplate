//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package idempotency

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/usecase/boundary/clock"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/xerrors"
)

// ttl は、冪等性キーの保持期間（= リトライ許容窓）です。
const ttl = 24 * time.Hour

// Metrics は、冪等性判定結果の o11y カウンタです（operationID ラベルで分解）。
// 各メソッドは ctx を第 1 引数に取り、OTel exemplar（メトリクス→トレース相関）を維持します。
type Metrics interface {
	// IncHit は、completed のエントリへの再送（replay）を計上します。
	IncHit(ctx context.Context, operationID string)
	// IncMiss は、新規 claim 成立（businessFn 実行へ進む）を計上します。
	IncMiss(ctx context.Context, operationID string)
	// IncConflict は、処理中キーへの並行再送（409）を計上します。
	IncConflict(ctx context.Context, operationID string)
	// IncFingerprintMismatch は、同一キー別ボディの再利用（422）を計上します。
	IncFingerprintMismatch(ctx context.Context, operationID string)
	// IncClaimFailure は、ErrLockTimeout 以外の Claim 失敗を計上します。
	IncClaimFailure(ctx context.Context, operationID string)
	// IncCompleteFailure は、Complete 失敗（結果保存失敗）を計上します。
	IncCompleteFailure(ctx context.Context, operationID string)
}

// Deps は、Run が必要とする依存です。
type Deps struct {
	Txm   tx.Manager
	Store idempotencybndry.Store
	Clock clock.Clock
	// Metrics は任意。nil の場合はすべてのカウンタ操作が no-op になります。
	Metrics Metrics
}

type nopMetrics struct{}

// Run は、ctx に Idempotency-Key が無ければ businessFn を素通し実行し、ある場合は業務 tx 内で
// claim → businessFn → complete を 1 tx で行います。再送は replay、並行 claim は 409、指紋不一致は 422。
// successStatus は completed のエントリへ保存する HTTP ステータス、戻り値 replayed は再生したことを表します。
//
// 前提: 本関数は「成功時のステータスが常に successStatus 単一」の操作向けです（例: PostUsers は 201 のみ）。
// replay 時は保存済みの応答ボディ（T）のみを復元し、保存済み response_status は呼び出し側へ伝播しません。
func Run[T any](
	ctx context.Context,
	deps Deps,
	successStatus int32,
	businessFn func(ctx context.Context) (T, error),
) (T, bool, error) {
	req, ok := requestFromContext(ctx)
	if !ok || req.Scope == "" {
		res, err := businessFn(ctx)
		return res, false, err
	}

	var result T
	var replayed bool
	err := deps.Txm.Do(ctx, func(ctx context.Context) error {
		claimed, err := deps.Store.Claim(ctx, idempotencybndry.ClaimParams{
			Scope:       req.Scope,
			Key:         req.Key,
			Method:      req.Method,
			Path:        req.Path,
			Fingerprint: req.Fingerprint,
			ExpiresAt:   deps.Clock.Now().Add(ttl),
		})
		if err != nil {
			if xerrors.Is(err, idempotencybndry.ErrLockTimeout) {
				deps.metrics().IncConflict(ctx, req.OperationID)
				return xerrors.Wrap(apperror.ErrConflict, "idempotency key is being processed, retry later")
			}
			deps.metrics().IncClaimFailure(ctx, req.OperationID)
			return err
		}

		if !claimed {
			res, replay, derr := decideExisting[T](ctx, deps, req)
			if derr != nil {
				return derr
			}
			result, replayed = res, replay
			return nil
		}

		// 新規 claim 成立。業務処理を同一 tx で実行する（失敗は tx ロールバックで claim ごと解放）。
		deps.metrics().IncMiss(ctx, req.OperationID)
		res, err := businessFn(ctx)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(res)
		if err != nil {
			return xerrors.Join(apperror.ErrInternal, xerrors.Wrap(err, "failed to encode idempotent response"))
		}
		if err := deps.Store.Complete(ctx, idempotencybndry.CompleteParams{
			Scope:           req.Scope,
			Key:             req.Key,
			ResponseStatus:  successStatus,
			ResponsePayload: payload,
		}); err != nil {
			deps.metrics().IncCompleteFailure(ctx, req.OperationID)
			return err
		}
		result = res
		return nil
	})
	if err != nil {
		var zero T
		return zero, false, err
	}
	return result, replayed, nil
}

// decideExisting は、既存キー（claim 衝突）に対して replay / 409 / 422 を判定します。
func decideExisting[T any](
	ctx context.Context, deps Deps, req Request,
) (T, bool, error) {
	var zero T
	rec, err := deps.Store.Get(ctx, req.Scope, req.Key)
	if err != nil {
		return zero, false, err
	}
	if rec == nil {
		// claim 衝突直後にエントリが消えた稀なレース。後で再試行させる。
		deps.metrics().IncConflict(ctx, req.OperationID)
		return zero, false, xerrors.Wrap(apperror.ErrConflict, "idempotency key state unavailable, retry later")
	}

	if !bytes.Equal(rec.Fingerprint, req.Fingerprint) {
		deps.metrics().IncFingerprintMismatch(ctx, req.OperationID)
		return zero, false, xerrors.Wrap(apperror.ErrValidation, "idempotency key reused with a different request")
	}

	if rec.Status != idempotencybndry.StatusCompleted {
		deps.metrics().IncConflict(ctx, req.OperationID)
		return zero, false, xerrors.Wrap(apperror.ErrConflict, "idempotency key is being processed, retry later")
	}

	// completed → 保存済み DTO を復元して replay。
	var result T
	if err := json.Unmarshal(rec.ResponsePayload, &result); err != nil {
		return zero, false, xerrors.Join(apperror.ErrInternal, xerrors.Wrap(err, "failed to decode stored idempotent response"))
	}
	deps.metrics().IncHit(ctx, req.OperationID)
	return result, true, nil
}

func (d Deps) metrics() Metrics {
	if d.Metrics == nil {
		return nopMetrics{}
	}
	return d.Metrics
}

func (nopMetrics) IncHit(context.Context, string)                 {}
func (nopMetrics) IncMiss(context.Context, string)                {}
func (nopMetrics) IncConflict(context.Context, string)            {}
func (nopMetrics) IncFingerprintMismatch(context.Context, string) {}
func (nopMetrics) IncClaimFailure(context.Context, string)        {}
func (nopMetrics) IncCompleteFailure(context.Context, string)     {}
