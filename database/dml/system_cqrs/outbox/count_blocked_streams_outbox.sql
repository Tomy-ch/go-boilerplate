-- name: CountBlockedStreamsOutbox :one
-- 先頭が dead のストリーム数（blocked stream の定義は docs/design/outbox.md の用語集）。
SELECT COUNT(*)
FROM (
    SELECT DISTINCT ON (ordering_key) status
    FROM outbox
    WHERE delivery_channel = $1
        AND ordering_key IS NOT NULL
        AND status <> 'published'
    ORDER BY ordering_key, ordering_sequence
) AS heads
WHERE heads.status = 'dead';
