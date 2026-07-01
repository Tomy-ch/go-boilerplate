-- name: MarkOutboxDead :execrows
-- attempts が max に達した恒久失敗行を dead へ遷移する。無限リトライを止め、手動 replay 対象として残置する。
UPDATE outbox
SET status = 'dead'
WHERE id = $1
    AND status = 'pending';
