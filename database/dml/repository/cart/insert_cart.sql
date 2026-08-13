-- name: CreateCart :exec
-- カートを新規登録する。user_id / session_token の一意制約違反は呼び出し側が衝突として扱う。
INSERT INTO carts (
    id,
    user_id,
    session_token,
    expires_at
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('user_id'),
    sqlc.arg('session_token'),
    sqlc.arg('expires_at')
);

-- name: CreateCartItem :exec
-- カート明細を 1 件登録する。同一カート内の同一商品は一意（cart_items_cart_id_product_id_unique）。
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
);
