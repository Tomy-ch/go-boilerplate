-- name: GetProductByID :one
-- ID から公開状態を問わない単一商品を取得します。
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
-- 公開中のみを返す GetPublishedProductByID とは可視範囲が異なり、未公開商品も返します
-- （用途は docs/spec/product/domain.md の Product.FindByID を参照）。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.id = sqlc.arg('product_id_param');
