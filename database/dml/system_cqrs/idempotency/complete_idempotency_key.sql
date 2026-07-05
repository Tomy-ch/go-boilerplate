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
