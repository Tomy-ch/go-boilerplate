-- name: CountBlockedStreamsOutbox :one
-- 先頭（最小の未 published sequence）が dead のストリーム数。head が dead のストリームは
-- head-of-line 規則により後続が claim されないため、復旧が要る対象として数える。
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
