-- name: GetIdempotencyKey :one
-- scope 必須（越境防止）。claim 衝突後の replay/409/422 判定に使う。
SELECT
    status,
    response_status,
    response_payload,
    request_fingerprint
FROM idempotency_keys
WHERE scope = $1
    AND idempotency_key = $2;
