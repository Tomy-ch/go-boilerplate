
-- === source: database/dml/system_cqrs/outbox/claim_pending_outbox.sql ===
-- name: ClaimPendingOutbox :many
-- pending 行を最大 $1 件 claim する。SKIP LOCKED により多インスタンスでも同一行を二重取得しない。
-- 順序保証は捨てているため id 昇順で十分。呼び出し側 tx の保持中だけロックされる。
SELECT
    id,
    message_id,
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    headers,
    attempts
FROM outbox
WHERE status = 'pending'
ORDER BY id
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- === source: database/dml/system_cqrs/outbox/delete_published_outbox.sql ===
-- name: DeletePublishedOutbox :execrows
-- retention 用 GC。published_at が cutoff より古い行を最大 $2 件削除し、削除件数を返す。
DELETE FROM outbox
WHERE id IN (
        SELECT o.id
        FROM outbox AS o
        WHERE o.status = 'published'
            AND o.published_at < $1
        ORDER BY o.published_at
        LIMIT $2
    );

-- === source: database/dml/system_cqrs/outbox/insert_outbox.sql ===
-- name: InsertOutbox :one
-- 業務 tx 内で outbox 行を 1 行 INSERT する（emit）。message_id は DB が採番し返す。
INSERT INTO outbox (
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    headers
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, message_id;

-- === source: database/dml/system_cqrs/outbox/mark_outbox_dead.sql ===
-- name: MarkOutboxDead :execrows
-- attempts が max に達した行を dead へ遷移する（dead の意味は docs/design/outbox.md）。
UPDATE outbox
SET status = 'dead'
WHERE id = $1
    AND status = 'pending';

-- === source: database/dml/system_cqrs/outbox/mark_outbox_failed.sql ===
-- name: MarkOutboxFailed :one
-- publish 失敗時に attempts を加算し last_error を記録する。加算後の attempts を返し、
-- 呼び出し側が max 到達判定（dead 化）に用いる。
UPDATE outbox
SET
    attempts = attempts + 1,
    last_error = $2
WHERE id = $1
    AND status = 'pending'
RETURNING attempts;

-- === source: database/dml/system_cqrs/outbox/mark_outbox_published.sql ===
-- name: MarkOutboxPublished :execrows
-- publish 成功行を published へ遷移する。published_at に遷移時刻を記録する。
UPDATE outbox
SET
    status = 'published',
    published_at = NOW()
WHERE id = $1
    AND status = 'pending';

-- === source: database/dml/system_cqrs/outbox/oldest_pending_outbox.sql ===
-- name: OldestPendingOutbox :one
-- SLI(outbox lag) 算出用。最古 pending 行の created_at を返す。pending 行が無ければ 0 行を返す。
SELECT created_at
FROM outbox
WHERE status = 'pending'
ORDER BY id
LIMIT 1;

-- === source: database/dml/system_cqrs/outbox/replay_dead_outbox.sql ===
-- name: ReplayDeadOutbox :execrows
-- dead 行を pending へ戻し再 publish 対象に復帰させる（運用 replay）。
-- $1 が NULL の場合は全 dead 行、指定時は当該 message_id のみを対象とする。
UPDATE outbox
SET
    status = 'pending',
    attempts = 0,
    last_error = NULL
WHERE status = 'dead'
    AND (sqlc.narg('message_id')::UUID IS NULL OR message_id = sqlc.narg('message_id')::UUID);
