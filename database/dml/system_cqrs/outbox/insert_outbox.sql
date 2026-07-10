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
