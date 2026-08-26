-- name: ListProductsByIDs :many
-- ID の集合から公開状態を問わない商品群を取得します。
-- 行はロックしません（カートは在庫を押さえない — docs/spec/cart/domain.md の Notes）。
-- 不存在の ID は結果に現れないため、返る件数は引数より少なくなり得ます。
-- 公開状態で絞らないのは、非公開化された明細に unpublished を立てる判定材料が要るためです。
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.id = ANY(@product_ids_param::UUID[])
ORDER BY p.id;
