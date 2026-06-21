
-- === source: database/dml/system_query/idempotency/claim_idempotency_key.sql ===
-- name: ClaimIdempotencyKey :one
-- 業務 tx 内で claimed 行を作る。既存キーがあれば ON CONFLICT DO NOTHING で 0 行を返す（= 既存キーあり）。
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

-- === source: database/dml/system_query/idempotency/complete_idempotency_key.sql ===
-- name: CompleteIdempotencyKey :execrows
-- 同一 tx 内で claimed → completed へ遷移し、結果 DTO(JSON) と HTTP ステータスを保存する。scope 必須（越境防止）。
UPDATE idempotency_keys
SET
    status = 'completed',
    response_status = $3,
    response_payload = $4,
    completed_at = NOW()
WHERE scope = $1
    AND idempotency_key = $2
    AND status = 'claimed';

-- === source: database/dml/system_query/idempotency/delete_expired_idempotency_keys.sql ===
-- name: DeleteExpiredIdempotencyKeys :execrows
-- TTL 失効行をバッチ削除する（$2 件ずつ）。一括 DELETE の long lock を避けるため GC ジョブが 0 件になるまで反復する。
DELETE FROM idempotency_keys
WHERE id IN (
        SELECT ik.id
        FROM idempotency_keys AS ik
        WHERE ik.expires_at < $1
        ORDER BY ik.expires_at
        LIMIT $2
    );

-- === source: database/dml/system_query/idempotency/get_idempotency_key.sql ===
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
