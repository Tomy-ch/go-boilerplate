-- name: CountDiscontinueImpactUsers :one
-- クーポンの受給対象になる確定済みユーザーの数を返す。
-- 実行側の SelectDiscontinueCouponRecipients と同じ条件を持つ。片方だけを変えてはならない。
-- 除外対象と見積もりの古さは docs/spec/usecase/product.md の GetDiscontinueImpact を参照。
SELECT COUNT(DISTINCT c.user_id)
FROM cart_items AS ci
INNER JOIN carts AS c ON c.id = ci.cart_id
INNER JOIN users AS u ON u.id = c.user_id
WHERE ci.product_id = sqlc.arg('product_id')
    AND u.deleted_at IS NULL;
