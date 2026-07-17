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
