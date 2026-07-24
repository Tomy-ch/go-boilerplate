-- name: ListPurchasesFeedFirst :many
-- 指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で先頭ページ取得する。
-- ステータス名は購入ステータスマスタとの結合で解決する（購入集約に属する固定参照マスタへの
-- 一意な等結合であり、単一集約の read）。一覧は概要のみで明細は含まない。
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
ORDER BY p.ordered_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');

-- name: ListPurchasesFeedAfter :many
-- (ordered_at DESC, id DESC) の keyset 境界より過去の購入履歴を返す。境界は直前ページ末尾行の
-- (ordered_at, id) で、ordered_at 同値は id で安定にタイブレークする。
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
ORDER BY p.ordered_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');
