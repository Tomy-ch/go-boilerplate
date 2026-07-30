-- name: GetProductByIDForUpdate :one
-- ID から公開状態を問わない単一商品を、更新のために悲観ロック（FOR UPDATE）して取得します。
-- 同一商品への並行書き込み（購入の在庫減算・在庫補充）を行ロックで直列化します。
-- ロック対象は products のみで、結合する固定参照マスタはロックしません（FOR UPDATE OF p）。
-- status_name / category_name は商品の付随表示値。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.id = sqlc.arg('product_id_param')
FOR UPDATE OF p;
