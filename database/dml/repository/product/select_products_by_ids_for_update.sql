-- name: ListProductsByIDsForUpdate :many
-- ID の集合から公開状態を問わない商品群を、更新のために悲観ロック（FOR UPDATE）して取得します。
-- ロック順序を id 昇順に固定することで、複数商品を同時にロックする処理同士のデッドロックを構造的に避けます（ADR-0033 (ordered-pessimistic-row-locks)）。
-- 不存在の ID は結果に現れないため、返る件数は引数より少なくなり得ます。
-- ロック対象は products のみで、結合する固定参照マスタはロックしません（FOR UPDATE OF p）。
-- status_name / category_name は商品の付随表示値。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.id = ANY(@product_ids_param::UUID [])
ORDER BY p.id
FOR UPDATE OF p;
