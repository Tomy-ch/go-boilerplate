-- name: CountDiscontinueImpactUsers :one
-- クーポンの受給対象になる確定済みユーザーの数を返す。
-- 実行側の SelectDiscontinueCouponRecipients と同じ条件を持つ。片方だけを変えてはならない。
-- ゲストのカートと退会済みユーザーを除くため、CountDiscontinueImpactCarts 以下になる。
-- 行はロックしないため、返した値は返した瞬間から古くなる。
SELECT COUNT(DISTINCT c.user_id)
FROM cart_items AS ci
INNER JOIN carts AS c ON c.id = ci.cart_id
INNER JOIN users AS u ON u.id = c.user_id
WHERE ci.product_id = sqlc.arg('product_id')
    AND u.deleted_at IS NULL;
