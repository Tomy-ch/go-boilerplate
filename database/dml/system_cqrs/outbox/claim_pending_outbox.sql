-- name: ClaimPendingOutbox :many
-- 指定チャネルの pending 行を最大 $2 件 claim する。SKIP LOCKED により多インスタンスでも同一行を二重取得しない。
-- バックオフ中（next_attempt_at が未来）の行は述語段階で外れるためロックもされず、SKIP LOCKED と干渉しない。
-- NOT EXISTS は head-of-line 規則。同一 ordering_key に未 published の先行 sequence がある行は claim しない。
-- 先行行が他インスタンスに claim されている間もその行は pending のままなので、ロックを SKIP して順序を飛ばすことはない。
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
        SELECT 1
        FROM outbox AS prior
        WHERE prior.ordering_key = o.ordering_key
            AND prior.ordering_sequence < o.ordering_sequence
            AND prior.status <> 'published'
    )
ORDER BY o.id
LIMIT $2
FOR UPDATE OF o SKIP LOCKED;
