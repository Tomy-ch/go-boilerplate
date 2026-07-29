
-- === source: database/dml/query_service/product/select_product_ranking.sql ===
-- name: ListProductRanking :many
-- 購入明細を商品単位で集計し、販売数量の降順で上位 limit_count 件を返します。
-- 公開済み（published_at 非 NULL）の商品のみを対象とし、非公開・未存在は集計から除外します。
-- キャンセル済み（canceled_at 設定済み）の購入は除外し、未払いの購入は含みます。
-- filter_by_period=true の場合は注文日時が ordered_after 以降の購入のみを集計対象にします（period=30d 用）。
-- 同一販売数量は商品 ID の昇順で安定的に並べます。
SELECT
    p.id AS product_id,
    p.name,
    p.price,
    SUM(pd.quantity)::BIGINT AS sold_quantity
FROM purchase_details AS pd
INNER JOIN purchases AS pur ON pd.purchase_id = pur.id AND pur.canceled_at IS NULL
INNER JOIN products AS p ON pd.product_id = p.id AND p.published_at IS NOT NULL
WHERE
    NOT sqlc.arg('filter_by_period')::BOOLEAN
    OR pur.ordered_at >= sqlc.narg('ordered_after')
GROUP BY p.id
ORDER BY sold_quantity DESC, p.id ASC
LIMIT sqlc.arg('limit_count');
