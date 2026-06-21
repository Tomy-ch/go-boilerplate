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
