-- name: DeletePublishedOutbox :execrows
-- retention 用 GC。published_at が cutoff より古い行を最大 $2 件削除し、削除件数を返す。
DELETE FROM outbox
WHERE id IN (
        SELECT o.id
        FROM outbox AS o
        WHERE o.status = 'published'
            AND o.published_at < $1
        ORDER BY o.published_at
        LIMIT $2
    );
