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
