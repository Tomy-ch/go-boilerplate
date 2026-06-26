-- name: OldestPendingOutbox :one
-- SLI(outbox lag) 算出用。最古 pending 行の created_at を返す。pending 行が無ければ 0 行を返す。
SELECT created_at
FROM outbox
WHERE status = 'pending'
ORDER BY id
LIMIT 1;
