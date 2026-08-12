-- name: MarkOutboxDead :execrows
-- attempts が max に達した行を dead へ遷移する（dead の意味は docs/design/outbox.md）。
UPDATE outbox
SET status = 'dead'
WHERE id = $1
    AND status = 'pending';
