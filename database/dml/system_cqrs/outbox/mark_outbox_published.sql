-- name: MarkOutboxPublished :execrows
-- publish 成功行を published へ遷移する。published_at に遷移時刻を記録する。
UPDATE outbox
SET
    status = 'published',
    published_at = NOW()
WHERE id = $1
    AND status = 'pending';
