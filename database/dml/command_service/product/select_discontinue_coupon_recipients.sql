-- name: SelectDiscontinueCouponRecipients :many
-- 廃番対象の商品を明細に持つカートのうち、所有者が確定していて退会もしていないユーザーを重複なく返す。
-- 絞り込みの理由と母集団が確定する時点は docs/spec/usecase/product.md の
-- Workflow — DiscontinueProduct の invariants を参照。
SELECT DISTINCT c.user_id::UUID AS user_id
FROM cart_items AS ci
INNER JOIN carts AS c ON c.id = ci.cart_id
INNER JOIN users AS u ON u.id = c.user_id
WHERE ci.product_id = sqlc.arg('product_id')
    AND u.deleted_at IS NULL;
