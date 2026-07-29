-- name: LockProductsForUpdate :many
-- 指定商品を ID 昇順に悲観ロック（FOR UPDATE）し、価格・在庫を返す。
-- ロック順序を id 昇順に固定することで複数商品購入同士のデッドロックを構造的に避ける（ADR-0100）。
SELECT
    p.id,
    p.price,
    p.quantity
FROM products AS p
WHERE p.id = ANY(@product_ids::UUID [])
ORDER BY p.id
FOR UPDATE;
