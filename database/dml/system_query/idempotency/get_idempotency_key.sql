-- name: GetIdempotencyKey :one
-- scope 必須（越境防止）。scope と idempotency_key で一致する行を返す。
SELECT
    status,
    response_status,
    response_payload,
    request_fingerprint
FROM idempotency_keys
WHERE scope = $1
    AND idempotency_key = $2;
