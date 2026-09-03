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

// Insert は、outbox 行を 1 行 INSERT して採番された message_id を返します。Headers が空なら '{}'、
// OrderingKey が空なら ordering_key / ordering_sequence を NULL で書きます。
func (s *store) Insert(ctx context.Context, p outboxbndry.EmitParams) (uuid.UUID, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	headers := p.Headers
	if len(headers) == 0 {
		headers = []byte("{}")
	}

	var orderingKey *string
	var orderingSequence *int64
	if p.OrderingKey != "" {
		orderingKey = &p.OrderingKey
		orderingSequence = &p.OrderingSequence
	}

	db := gen.New(driver.New(ctx, s.db))
	row, err := db.InsertOutbox(ctx, &gen.InsertOutboxParams{
		AggregateType:    p.AggregateType,
		AggregateID:      p.AggregateID,
		EventType:        p.EventType,
		Payload:          p.Payload,
		Headers:          headers,
		DeliveryChannel:  p.Channel.String(),
		OrderingKey:      orderingKey,
		OrderingSequence: orderingSequence,
	})
	if err != nil {
		return uuid.UUID{}, pgerror.NormalizeError(err)
	}
	return row.MessageID, nil
}

// ClaimPending は、channel の pending 行のうち next_attempt_at が到来し、同じ ordering_key に未 publish の
// 先行 ordering_sequence が無い行を id 順に最大 limit 件、FOR UPDATE SKIP LOCKED で取得します。
func (s *store) ClaimPending(
	ctx context.Context,
	channel outboxbndry.Channel,
	limit int32,
) ([]outboxbndry.PendingMessage, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.ClaimPendingOutbox(ctx, &gen.ClaimPendingOutboxParams{
		DeliveryChannel: channel.String(),
		Limit:           limit,
	})
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
			CreatedAt: row.CreatedAt,
		})
	}
	return messages, nil
}

// MarkPublished は、status = 'pending' の行だけを published へ遷移し published_at を NOW() にします。
// pending でなければ 0 行更新で成功します。
func (s *store) MarkPublished(ctx context.Context, id int64) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	if _, err := db.MarkOutboxPublished(ctx, id); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// MarkFailed は、status = 'pending' の行だけ attempts を +1 し、last_error と next_attempt_at を書き換えます。
// pending でなければ 0 行更新で成功します。
func (s *store) MarkFailed(ctx context.Context, id int64, lastErr string, nextAttemptAt time.Time) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	if _, err := db.MarkOutboxFailed(ctx, &gen.MarkOutboxFailedParams{
		ID:            id,
		LastError:     &lastErr,
		NextAttemptAt: nextAttemptAt,
	}); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// MarkDead は、status = 'pending' の行だけを dead へ遷移します。pending でなければ 0 行更新で成功します。
func (s *store) MarkDead(ctx context.Context, id int64) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	if _, err := db.MarkOutboxDead(ctx, id); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// ReplayDead は、dead 行を pending へ戻して attempts = 0、last_error = NULL、next_attempt_at = NOW() にし、
// 更新件数を返します。messageID が nil なら全 dead 行が対象です。
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

// DeletePublished は、published_at が cutoff より古い published 行を published_at の古い順に limit 件まで
// 削除し、削除件数を返します。
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

// OldestPendingCreatedAt は、channel の pending 行を id 順に 1 件読んで created_at を返します。
// pgx.ErrNoRows は ok=false に写します。バックオフ中の行も含みます。
func (s *store) OldestPendingCreatedAt(
	ctx context.Context,
	channel outboxbndry.Channel,
) (time.Time, bool, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	createdAt, err := db.OldestPendingOutbox(ctx, channel.String())
	if err != nil {
		if xerrors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, pgerror.NormalizeError(err)
	}
	return createdAt, true, nil
}

// CountBlockedStreams は、channel の未 publish 行を ordering_key ごとに ordering_sequence 最小の 1 行
// （DISTINCT ON）に絞り、その status が dead の ordering_key を数えます。
func (s *store) CountBlockedStreams(ctx context.Context, channel outboxbndry.Channel) (int64, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	count, err := db.CountBlockedStreamsOutbox(ctx, channel.String())
	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}
	return count, nil
}
