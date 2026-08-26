
-- === source: database/dml/system_cqrs/idempotency/claim_idempotency_key.sql ===
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

-- === source: database/dml/system_cqrs/idempotency/complete_idempotency_key.sql ===
-- name: CompleteIdempotencyKey :execrows
-- 同一 tx 内で claimed → completed へ遷移し、結果 DTO(JSON) と HTTP ステータスを保存する。scope 必須。
UPDATE idempotency_keys
SET
    status = 'completed',
    response_status = $3,
    response_payload = $4,
    completed_at = NOW()
WHERE scope = $1
    AND idempotency_key = $2
    AND status = 'claimed';

-- === source: database/dml/system_cqrs/idempotency/delete_expired_idempotency_keys.sql ===
-- name: DeleteExpiredIdempotencyKeys :execrows
-- TTL 失効行を最大 $2 件削除し、削除件数を返す。
DELETE FROM idempotency_keys
WHERE id IN (
        SELECT ik.id
        FROM idempotency_keys AS ik
        WHERE ik.expires_at < $1
        ORDER BY ik.expires_at
        LIMIT $2
    );

-- === source: database/dml/system_cqrs/idempotency/get_idempotency_key.sql ===
-- name: GetIdempotencyKey :one
-- scope 必須。scope と idempotency_key で一致する行を返す。
SELECT
    status,
    response_status,
    response_payload,
    request_fingerprint
FROM idempotency_keys
WHERE scope = $1
    AND idempotency_key = $2;
