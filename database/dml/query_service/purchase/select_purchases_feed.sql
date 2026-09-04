-- name: ListPurchasesFeedFirst :many
-- 指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で先頭ページ取得する。
-- ページを CTE で先に閉じてから結合するのは、明細の要約を解決する LATERAL が LIMIT 前の候補行すべてに
-- 対して評価されるのを防ぐため。
-- 明細 1 件以上は Purchase 集約の生成不変条件（docs/spec/domain/purchase.md）のため、LATERAL は INNER で結合する。
-- 注文日時は半開区間 [ordered_after, ordered_before)（internal/usecase/tools/timewindow/README.md）で絞り込む。
WITH page AS (
    SELECT
        p.id,
        p.code,
        p.total_amount,
        p.ordered_at,
        p.status_id
    FROM purchases AS p
    WHERE p.user_id = sqlc.arg('user_id')
        AND (
            p.ordered_at >= sqlc.narg('ordered_after')
            OR sqlc.narg('ordered_after') IS NULL
        )
        AND (
            p.ordered_at < sqlc.narg('ordered_before')
            OR sqlc.narg('ordered_before') IS NULL
        )
        AND (
            sqlc.narg('status_codes')::SMALLINT[] IS NULL
            OR p.status_id IN (
                SELECT s.id FROM purchase_statuses AS s
                WHERE s.code = ANY(sqlc.narg('status_codes')::SMALLINT[])
            )
        )
    ORDER BY p.ordered_at DESC, p.id DESC
    LIMIT sqlc.arg('limit_param')
)

SELECT
    page.id,
    page.code,
    page.total_amount,
    page.ordered_at,
    ps.id AS status_id,
    ps.code AS status_code,
    ps.name AS status_name,
    first_item.product_name AS first_item_name,
    item_agg.item_count
FROM page
INNER JOIN purchase_statuses AS ps ON page.status_id = ps.id
INNER JOIN LATERAL (
    SELECT pr.name AS product_name
    FROM purchase_details AS d
    INNER JOIN products AS pr ON d.product_id = pr.id
    WHERE d.purchase_id = page.id
    ORDER BY d.id
    LIMIT 1
) AS first_item ON TRUE
INNER JOIN LATERAL (
    SELECT COUNT(*)::BIGINT AS item_count
    FROM purchase_details AS d
    WHERE d.purchase_id = page.id
) AS item_agg ON TRUE
ORDER BY page.ordered_at DESC, page.id DESC;

-- name: ListPurchasesFeedAfter :many
-- (ordered_at DESC, id DESC) の keyset 境界より過去の購入履歴を返す。境界は直前ページ末尾行の
-- (ordered_at, id) で、ordered_at 同値は id で安定にタイブレークする。
-- 期間とステータスの絞り込みは先頭ページと同一条件で、ページ送りの間も呼び出し側が同じ条件を渡す前提である。
-- ページを CTE で閉じてから要約を結合する形も先頭ページと同一。
WITH page AS (
    SELECT
        p.id,
        p.code,
        p.total_amount,
        p.ordered_at,
        p.status_id
    FROM purchases AS p
    WHERE p.user_id = sqlc.arg('user_id')
        AND (
            p.ordered_at < sqlc.arg('after_ordered_at')
            OR (p.ordered_at = sqlc.arg('after_ordered_at') AND p.id < sqlc.arg('after_id'))
        )
        AND (
            p.ordered_at >= sqlc.narg('ordered_after')
            OR sqlc.narg('ordered_after') IS NULL
        )
        AND (
            p.ordered_at < sqlc.narg('ordered_before')
            OR sqlc.narg('ordered_before') IS NULL
        )
        AND (
            sqlc.narg('status_codes')::SMALLINT[] IS NULL
            OR p.status_id IN (
                SELECT s.id FROM purchase_statuses AS s
                WHERE s.code = ANY(sqlc.narg('status_codes')::SMALLINT[])
            )
        )
    ORDER BY p.ordered_at DESC, p.id DESC
    LIMIT sqlc.arg('limit_param')
)

SELECT
    page.id,
    page.code,
    page.total_amount,
    page.ordered_at,
    ps.id AS status_id,
    ps.code AS status_code,
    ps.name AS status_name,
    first_item.product_name AS first_item_name,
    item_agg.item_count
FROM page
INNER JOIN purchase_statuses AS ps ON page.status_id = ps.id
INNER JOIN LATERAL (
    SELECT pr.name AS product_name
    FROM purchase_details AS d
    INNER JOIN products AS pr ON d.product_id = pr.id
    WHERE d.purchase_id = page.id
    ORDER BY d.id
    LIMIT 1
) AS first_item ON TRUE
INNER JOIN LATERAL (
    SELECT COUNT(*)::BIGINT AS item_count
    FROM purchase_details AS d
    WHERE d.purchase_id = page.id
) AS item_agg ON TRUE
ORDER BY page.ordered_at DESC, page.id DESC;
