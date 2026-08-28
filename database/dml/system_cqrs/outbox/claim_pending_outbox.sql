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
    o.attempts
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
