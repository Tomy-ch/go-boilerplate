
-- === source: database/dml/system_cqrs/outbox/claim_pending_outbox.sql ===
-- name: ClaimPendingOutbox :many
-- 指定チャネルの pending 行を最大 $2 件、SKIP LOCKED で claim する（多重取得防止は ADR-0056）。
-- バックオフ中（next_attempt_at が未来）の行は述語段階で外れるためロックもされず、SKIP LOCKED と干渉しない。
-- NOT EXISTS は head-of-line 規則（ADR-0072）。同一 ordering_key の先行 sequence が未 published なら claim しない。
-- ordering_key が NULL の行は NULL 比較で NOT EXISTS が真になり、順序を持たないチャネルは除外されない。
SELECT
    o.id,
    o.message_id,
    o.aggregate_type,
    o.aggregate_id,
    o.event_type,
    o.payload,
    o.headers,
    o.attempts,
    o.created_at
FROM outbox AS o
WHERE o.status = 'pending'
    AND o.delivery_channel = $1
    AND o.next_attempt_at <= NOW()
    AND NOT EXISTS (
        SELECT earlier.id
        FROM outbox AS earlier
        WHERE earlier.ordering_key = o.ordering_key
            AND earlier.ordering_sequence < o.ordering_sequence
            AND earlier.status <> 'published'
    )
ORDER BY o.id
LIMIT $2
FOR UPDATE OF o SKIP LOCKED;

-- === source: database/dml/system_cqrs/outbox/count_blocked_streams_outbox.sql ===
-- name: CountBlockedStreamsOutbox :one
-- 先頭が dead のストリーム数（blocked stream の定義は docs/design/outbox.md の用語集）。
SELECT COUNT(*)
FROM (
    SELECT DISTINCT ON (ordering_key) status
    FROM outbox
    WHERE delivery_channel = $1
        AND ordering_key IS NOT NULL
        AND status <> 'published'
    ORDER BY ordering_key, ordering_sequence
) AS heads
WHERE heads.status = 'dead';

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
-- delivery_channel は必須（既定値なし。EmitParams.Channel 参照）。
INSERT INTO outbox (
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    headers,
    delivery_channel,
    ordering_key,
    ordering_sequence
) VALUES (
    $1, $2, $3, $4, $5, $6, sqlc.narg('ordering_key'), sqlc.narg('ordering_sequence')
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
-- name: MarkOutboxFailed :execrows
-- publish 失敗時に last_error を記録し、次に claim してよい時刻をバックオフ後の時刻へ進める。
-- attempts は診断のために加算し続けるが、dead 判定の基準ではない（判定はエラー分類。ADR-0058）。
UPDATE outbox
SET
    attempts = attempts + 1,
    last_error = $2,
    next_attempt_at = $3
WHERE id = $1
    AND status = 'pending';

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
-- SLI(outbox lag) 算出用。指定チャネルの最古 pending 行の created_at を返す。pending 行が無ければ 0 行を返す。
-- バックオフ中の行も未配送なので除外しない（lag は未配送の最古行の年齢）。
SELECT created_at
FROM outbox
WHERE status = 'pending'
    AND delivery_channel = $1
ORDER BY id
LIMIT 1;

-- === source: database/dml/system_cqrs/outbox/replay_dead_outbox.sql ===
-- name: ReplayDeadOutbox :execrows
-- dead 行を pending へ戻し再 publish 対象に復帰させる（運用 replay）。
-- $1 が NULL の場合は全 dead 行、指定時は当該 message_id のみを対象とする。
-- next_attempt_at を現在時刻へ戻さないと、dead 化前のバックオフ済み時刻が残って直後に claim されない。
UPDATE outbox
SET
    status = 'pending',
    attempts = 0,
    last_error = NULL,
    next_attempt_at = NOW()
WHERE status = 'dead'
    AND (sqlc.narg('message_id')::UUID IS NULL OR message_id = sqlc.narg('message_id')::UUID);
