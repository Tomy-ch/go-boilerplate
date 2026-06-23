//go:generate mockgen -source=$GOFILE -destination=mock/mock_metrics.gen.go -package=mock_$GOPACKAGE
package idempotency

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/usecase/boundary/clock"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/xerrors"
)

// ttl は、冪等性キーの保持期間（= リトライ許容窓）です。
const ttl = 24 * time.Hour

// Metrics は、冪等性の o11y カウンタです（operationID ラベルで分解）。
type Metrics interface {
	IncReplay(operationID string)
	IncConflict(operationID string)
	IncFingerprintMismatch(operationID string)
}

// Deps は、Run が必要とする依存です。
type Deps struct {
	Txm   tx.Manager
	Store idempotencybndry.Store
	Clock clock.Clock
	// Metrics は任意。nil の場合カウンタは no-op です（観測性バックエンド配線時に実装を注入する）。
	Metrics Metrics
}

type nopMetrics struct{}

// Run は、ctx に Idempotency-Key が無ければ businessFn を素通し実行し、ある場合は業務 tx 内で
// claim → businessFn → complete を 1 tx で行います。再送は replay、並行 claim は 409、指紋不一致は 422。
// successStatus は completed 行へ保存する HTTP ステータス、戻り値 replayed は再生したことを表します。
//
// 前提: 本関数は「成功時のステータスが常に successStatus 単一」の操作向けです（例: PostUsers は 201 のみ）。
// replay 時は保存済みの応答ボディ（T）のみを復元し、保存済み response_status は呼び出し側へ伝播しません。
func Run[T any](
	ctx context.Context,
	deps Deps,
	successStatus int,
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
			if errors.Is(err, idempotencybndry.ErrLockTimeout) {
				deps.metrics().IncConflict(req.OperationID)
				return xerrors.Wrap(apperror.ErrConflict, "idempotency key is being processed, retry later")
			}
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
		res, err := businessFn(ctx)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(res)
		if err != nil {
			return xerrors.Wrap(apperror.ErrInternal, "failed to encode idempotent response: "+err.Error())
		}
		if err := deps.Store.Complete(ctx, idempotencybndry.CompleteParams{
			Scope:           req.Scope,
			Key:             req.Key,
			ResponseStatus:  int32(successStatus), //nolint:gosec // HTTP ステータスコードは int32 に収まる
			ResponsePayload: payload,
		}); err != nil {
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
		// claim 衝突直後に行が消えた稀なレース。後で再試行させる。
		return zero, false, xerrors.Wrap(apperror.ErrConflict, "idempotency key state unavailable, retry later")
	}

	if !bytes.Equal(rec.Fingerprint, req.Fingerprint) {
		deps.metrics().IncFingerprintMismatch(req.OperationID)
		return zero, false, xerrors.Wrap(apperror.ErrValidation, "idempotency key reused with a different request")
	}

	if rec.Status != idempotencybndry.StatusCompleted {
		deps.metrics().IncConflict(req.OperationID)
		return zero, false, xerrors.Wrap(apperror.ErrConflict, "idempotency key is being processed, retry later")
	}

	// completed → 保存済み DTO を復元して replay。
	var result T
	if err := json.Unmarshal(rec.ResponsePayload, &result); err != nil {
		return zero, false, xerrors.Wrap(apperror.ErrInternal, "failed to decode stored idempotent response: "+err.Error())
	}
	deps.metrics().IncReplay(req.OperationID)
	return result, true, nil
}

func (d Deps) metrics() Metrics {
	if d.Metrics == nil {
		return nopMetrics{}
	}
	return d.Metrics
}

func (nopMetrics) IncReplay(string)              {}
func (nopMetrics) IncConflict(string)            {}
func (nopMetrics) IncFingerprintMismatch(string) {}
