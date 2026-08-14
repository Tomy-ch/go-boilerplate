-- name: ListPurchasesFeedFirst :many
-- 指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で先頭ページ取得する。
-- ステータス名は購入ステータスマスタとの結合で解決する（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
-- 一覧は概要のみで明細は含まない。
-- filter_by_period=true の場合は注文日時が半開区間 [ordered_after, ordered_before) の購入だけを返す。
SELECT
    p.id,
    p.code,
    p.total_amount,
    p.ordered_at,
    ps.id AS status_id,
    ps.name AS status_name
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.user_id = sqlc.arg('user_id')
    AND (
        NOT sqlc.arg('filter_by_period')::BOOLEAN
        OR (
            p.ordered_at >= sqlc.narg('ordered_after')
            AND p.ordered_at < sqlc.narg('ordered_before')
        )
    )
ORDER BY p.ordered_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');

-- name: ListPurchasesFeedAfter :many
-- (ordered_at DESC, id DESC) の keyset 境界より過去の購入履歴を返す。境界は直前ページ末尾行の
-- (ordered_at, id) で、ordered_at 同値は id で安定にタイブレークする。
-- 期間の絞り込みは先頭ページと同一条件で、ページ送りの間も呼び出し側が同じ期間を渡す前提である。
SELECT
    p.id,
    p.code,
    p.total_amount,
    p.ordered_at,
    ps.id AS status_id,
    ps.name AS status_name
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.user_id = sqlc.arg('user_id')
    AND (
        p.ordered_at < sqlc.arg('after_ordered_at')
        OR (p.ordered_at = sqlc.arg('after_ordered_at') AND p.id < sqlc.arg('after_id'))
    )
    AND (
        NOT sqlc.arg('filter_by_period')::BOOLEAN
        OR (
            p.ordered_at >= sqlc.narg('ordered_after')
            AND p.ordered_at < sqlc.narg('ordered_before')
        )
    )
ORDER BY p.ordered_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');
