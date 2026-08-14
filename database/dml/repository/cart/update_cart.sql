-- name: UpdateCart :execrows
-- カートの親行を更新し、更新件数を返す。0 件は対象が存在しないことを意味し、呼び出し側が NotFound
-- として扱う。所有者の確定（user_id のセットと session_token の NULL 化）もこの 1 本で反映される。
UPDATE carts
SET
    user_id = sqlc.arg('user_id'),
    session_token = sqlc.arg('session_token'),
    expires_at = sqlc.arg('expires_at'),
    updated_at = NOW()
WHERE carts.id = sqlc.arg('id');

-- name: UpsertCartItem :exec
-- 明細を商品ごとに登録または置換する。added_at は SET の対象に含めない。置換のたびに更新すると
-- 「いつカートへ入ったか」が失われ、切り捨ての順序を決める基準が壊れるため。
INSERT INTO cart_items (
    id,
    cart_id,
    product_id,
    quantity,
    last_seen_price,
    added_at
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('cart_id'),
    sqlc.arg('product_id'),
    sqlc.arg('quantity'),
    sqlc.arg('last_seen_price'),
    sqlc.arg('added_at')
)
ON CONFLICT ON CONSTRAINT cart_items_cart_id_product_id_unique DO UPDATE
    SET
        quantity = excluded.quantity,
        last_seen_price = excluded.last_seen_price,
        updated_at = NOW();

-- name: DeleteCartItemsNotIn :exec
-- 指定した商品の集合に含まれない明細を取り除く。集約が保持しなくなった明細を落とすためのもので、
-- 空の集合を渡した場合はそのカートの明細をすべて削除する。
DELETE FROM cart_items
WHERE cart_items.cart_id = sqlc.arg('cart_id')
    AND NOT (cart_items.product_id = ANY(sqlc.arg('product_ids')::UUID []));
