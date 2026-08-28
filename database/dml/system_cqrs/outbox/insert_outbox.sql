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
