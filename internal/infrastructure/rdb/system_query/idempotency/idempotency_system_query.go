// Package idempotency は、冪等性キーの永続化（boundary idempotency.Store）の RDB 実装を提供します。
package idempotency

import (
	"context"
	"errors"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5"
)

// claimLockTimeout は、claim 時に並行リクエストを待つロックタイムアウト上限です。
const claimLockTimeout = "3s"

type store struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、idempotency.Store の RDB 実装を生成して返します。
func New(
	provider driver.DatabaseDriver,
	tf observability.TracerFactory,
) idempotencybndry.Store {
	return &store{
		db:     provider,
		tracer: tf.Infra(),
	}
}

// Claim は、claimed 行を作ります。業務 tx 内から呼ばれる前提（SET LOCAL lock_timeout が効く）。
func (s *store) Claim(ctx context.Context, p idempotencybndry.ClaimParams) (bool, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	ldb := driver.New(ctx, s.db)
	if _, err := ldb.Exec(ctx, "SET LOCAL lock_timeout = '"+claimLockTimeout+"'"); err != nil {
		return false, pgerror.NormalizeError(err)
	}

	db := gen.New(ldb)
	_, err := db.ClaimIdempotencyKey(ctx, &gen.ClaimIdempotencyKeyParams{
		Scope:              p.Scope,
		IdempotencyKey:     p.Key,
		RequestMethod:      p.Method,
		RequestPath:        p.Path,
		RequestFingerprint: p.Fingerprint,
		ExpiresAt:          p.ExpiresAt,
	})
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, pgx.ErrNoRows):
		// ON CONFLICT DO NOTHING で 0 行 = 既存キーあり。
		return false, nil
	case pgerror.IsLockNotAvailable(err):
		return false, idempotencybndry.ErrLockTimeout
	default:
		return false, pgerror.NormalizeError(err)
	}
}

// Get は、(scope, key) の保存済み状態を取得します。存在しなければ nil, nil。
func (s *store) Get(ctx context.Context, scope, key string) (*idempotencybndry.Record, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	row, err := db.GetIdempotencyKey(ctx, &gen.GetIdempotencyKeyParams{
		Scope:          scope,
		IdempotencyKey: key,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil //nolint:nilnil // 「存在しない」を nil, nil で表す（呼び出し側が分岐）
		}
		return nil, pgerror.NormalizeError(err)
	}

	return &idempotencybndry.Record{
		Status:          idempotencybndry.Status(row.Status),
		ResponseStatus:  row.ResponseStatus,
		ResponsePayload: row.ResponsePayload,
		Fingerprint:     row.RequestFingerprint,
	}, nil
}

// Complete は、claimed → completed へ遷移し結果を保存します。
func (s *store) Complete(ctx context.Context, p idempotencybndry.CompleteParams) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	status := p.ResponseStatus
	affected, err := db.CompleteIdempotencyKey(ctx, &gen.CompleteIdempotencyKeyParams{
		Scope:           p.Scope,
		IdempotencyKey:  p.Key,
		ResponseStatus:  &status,
		ResponsePayload: p.ResponsePayload,
	})
	if err != nil {
		return pgerror.NormalizeError(err)
	}
	if affected == 0 {
		// 同一 tx で claim 済みの行が complete 対象にならない＝前提崩壊（404 ではなく内部エラー）。
		return xerrors.Wrap(apperror.ErrInternal, "complete: claimed row not found in the same transaction")
	}
	return nil
}

// DeleteExpired は、cutoff より古い行を limit 件まで削除し、削除件数を返します（GC）。
func (s *store) DeleteExpired(ctx context.Context, cutoff time.Time, limit int32) (int64, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	affected, err := db.DeleteExpiredIdempotencyKeys(ctx, &gen.DeleteExpiredIdempotencyKeysParams{
		ExpiresAt: cutoff,
		Limit:     limit,
	})
	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}
	return affected, nil
}
