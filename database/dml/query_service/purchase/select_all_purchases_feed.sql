-- name: ListAllPurchasesFeedFirst :many
-- 購入者を問わず購入履歴を (ordered_at DESC, id DESC) の安定順で先頭ページ取得する（admin の可視範囲）。
-- 所有権で閉じる ListPurchasesFeedFirst と母集団だけが異なり、並び順・要約の解決・期間とステータスの
-- 絞り込みは同一である。所有権の有無をクエリの別名で表し、呼び出し側の分岐なしに母集団が混ざらないようにする。
-- ページを CTE で先に閉じてから結合する理由、LATERAL を INNER で結合する理由は
-- database/dml/query_service/purchase/select_purchases_feed.sql と同じ。
WITH page AS (
    SELECT
        p.id,
        p.code,
        p.total_amount,
        p.ordered_at,
        p.status_id
    FROM purchases AS p
    WHERE (
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

-- name: ListAllPurchasesFeedAfter :many
-- 購入者を問わず (ordered_at DESC, id DESC) の keyset 境界より過去の購入履歴を返す（admin の可視範囲）。
-- 境界の解釈は ListPurchasesFeedAfter と同一で、母集団だけが異なる。
WITH page AS (
    SELECT
        p.id,
        p.code,
        p.total_amount,
        p.ordered_at,
        p.status_id
    FROM purchases AS p
    WHERE (
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
