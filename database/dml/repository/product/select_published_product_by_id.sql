-- name: GetPublishedProductByID :one
-- ID から公開中の単一商品を取得します。
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
-- 「公開中」を定義するのは Product.IsPublished で、以下の条件はその実行形です。片方だけ変更しないこと。
-- 非公開・未存在はいずれも該当行なし（0 行）で返ります。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.id = sqlc.arg('product_id_param')
    AND p.published_at IS NOT NULL;
