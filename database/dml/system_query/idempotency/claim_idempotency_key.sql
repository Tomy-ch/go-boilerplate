-- name: ClaimIdempotencyKey :one
-- 業務 tx 内でキーを claim する。既存キーがある場合は 0 行を返す。
INSERT INTO idempotency_keys (
    scope,
    idempotency_key,
    request_method,
    request_path,
    request_fingerprint,
    status,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, 'claimed', $6
)
ON CONFLICT ON CONSTRAINT idempotency_keys_scope_key_unique DO NOTHING
RETURNING id;
