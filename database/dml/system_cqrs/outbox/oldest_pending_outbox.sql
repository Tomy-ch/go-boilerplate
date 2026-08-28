-- name: OldestPendingOutbox :one
-- SLI(outbox lag) 算出用。指定チャネルの最古 pending 行の created_at を返す。pending 行が無ければ 0 行を返す。
-- バックオフ中の行も未配送なので除外しない（lag は未配送の最古行の年齢）。
SELECT created_at
FROM outbox
WHERE status = 'pending'
    AND delivery_channel = $1
ORDER BY id
LIMIT 1;
