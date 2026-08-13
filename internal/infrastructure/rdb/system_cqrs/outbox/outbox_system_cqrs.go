// Package outbox は、outbox テーブルの永続化（boundary outbox.Store）の RDB 実装を提供します。
package outbox

import (
	"context"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5"
)

type store struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、outbox.Store の RDB 実装を生成して返します。
func New(
	provider driver.DatabaseDriver,
	tf observability.TracerFactory,
) outboxbndry.Store {
	return &store{
		db:     provider,
		tracer: tf.Infra(),
	}
}

// Insert は、業務 tx 内で outbox 行を 1 行 INSERT し、採番された message_id を返します。
func (s *store) Insert(ctx context.Context, p outboxbndry.EmitParams) (uuid.UUID, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	headers := p.Headers
	if len(headers) == 0 {
		headers = []byte("{}")
	}

	db := gen.New(driver.New(ctx, s.db))
	row, err := db.InsertOutbox(ctx, &gen.InsertOutboxParams{
		AggregateType: p.AggregateType,
		AggregateID:   p.AggregateID,
		EventType:     p.EventType,
		Payload:       p.Payload,
		Headers:       headers,
	})
	if err != nil {
		return uuid.UUID{}, pgerror.NormalizeError(err)
	}
	return row.MessageID, nil
}

// ClaimPending は、pending 行を最大 limit 件ロックして返します。並行ワーカーが同一行を取得しません。
func (s *store) ClaimPending(ctx context.Context, limit int32) ([]outboxbndry.PendingMessage, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.ClaimPendingOutbox(ctx, limit)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	messages := make([]outboxbndry.PendingMessage, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, outboxbndry.PendingMessage{
			ID:        row.ID,
			MessageID: row.MessageID,
			EventType: row.EventType,
			Payload:   row.Payload,
			Headers:   row.Headers,
			Attempts:  row.Attempts,
		})
	}
	return messages, nil
}

// MarkPublished は、publish 成功行を published へ遷移します。
func (s *store) MarkPublished(ctx context.Context, id int64) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	if _, err := db.MarkOutboxPublished(ctx, id); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// MarkFailed は、attempts を加算し last_error を記録し、加算後の attempts を返します。
// 既に pending でない（並行処理で遷移済み）場合は 0 を返します。
func (s *store) MarkFailed(ctx context.Context, id int64, lastErr string) (int32, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	attempts, err := db.MarkOutboxFailed(ctx, &gen.MarkOutboxFailedParams{
		ID:        id,
		LastError: &lastErr,
	})
	if err != nil {
		if xerrors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, pgerror.NormalizeError(err)
	}
	return attempts, nil
}

// MarkDead は、行を dead へ遷移します。
func (s *store) MarkDead(ctx context.Context, id int64) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	if _, err := db.MarkOutboxDead(ctx, id); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// ReplayDead は、dead 行を pending へ戻し、戻した件数を返します（messageID が nil なら全 dead 行）。
func (s *store) ReplayDead(ctx context.Context, messageID *uuid.UUID) (int64, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	affected, err := db.ReplayDeadOutbox(ctx, messageID)
	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}
	return affected, nil
}

// DeletePublished は、published_at が cutoff より古い行を limit 件まで削除し、削除件数を返します（GC）。
func (s *store) DeletePublished(ctx context.Context, cutoff time.Time, limit int32) (int64, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	affected, err := db.DeletePublishedOutbox(ctx, &gen.DeletePublishedOutboxParams{
		PublishedAt: &cutoff,
		Limit:       limit,
	})
	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}
	return affected, nil
}

// OldestPendingCreatedAt は、最古 pending 行の created_at を返します（SLI=outbox lag 用）。
func (s *store) OldestPendingCreatedAt(ctx context.Context) (time.Time, bool, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	createdAt, err := db.OldestPendingOutbox(ctx)
	if err != nil {
		if xerrors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, pgerror.NormalizeError(err)
	}
	return createdAt, true, nil
}
