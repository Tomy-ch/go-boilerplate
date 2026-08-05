-- name: ListLowStockProducts :many
-- 在庫が警告閾値以下の商品を、在庫の少ない順（同数は ID 昇順）で最大 limit 件取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- stock_warning_threshold が NULL（閾値未設定）の商品は警告対象外として明示的に除外します。
-- 「在庫僅少」を定義するのは Product.IsLowStock で、以下の条件はその実行形です。片方だけ変更しないこと。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.stock_warning_threshold IS NOT NULL
    AND p.quantity <= p.stock_warning_threshold
ORDER BY p.quantity ASC, p.id ASC
LIMIT sqlc.arg('limit_param');
