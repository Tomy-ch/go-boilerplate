-- name: MarkOutboxFailed :one
-- publish 失敗時に attempts を加算し last_error を記録する。加算後の attempts を返し、
-- 呼び出し側が max 到達判定（dead 化）に用いる。次 poll で自然に再送される。
UPDATE outbox
SET
    attempts = attempts + 1,
    last_error = $2
WHERE id = $1
    AND status = 'pending'
RETURNING attempts;
