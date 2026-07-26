-- name: GetProductByID :one
-- ID から公開状態を問わない単一商品を取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- 公開中のみを返す GetPublishedProductByID とは可視範囲が異なり、未公開商品も返します
-- （公開日時の設定そのものを更新対象とする管理用途の read-modify-write に用います）。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.id = sqlc.arg('product_id_param');
