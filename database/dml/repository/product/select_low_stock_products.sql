-- name: ListLowStockProducts :many
-- 在庫が警告閾値以下の商品を、在庫の少ない順（同数は ID 昇順）で最大 limit 件取得します。
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
-- 閾値未設定（NULL）の商品は WHERE で明示的に除外する（意味は docs/spec/product/domain.md の
-- Product.FindAllLowStock を参照）。
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
