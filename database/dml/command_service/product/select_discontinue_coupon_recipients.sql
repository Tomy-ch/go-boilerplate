-- name: SelectDiscontinueCouponRecipients :many
-- 廃番対象の商品を明細に持つカートのうち、所有者が確定していて退会もしていないユーザーを重複なく返す。
-- ゲストのカート（owner_id が NULL）は所有者が居ないため受給者にならない。
-- 母集団はこの問い合わせが走った時点で確定し、以降にカートへ投入した利用者は含まれない
-- （理由は docs/spec/usecase/product.md の廃番）。
SELECT DISTINCT c.owner_id::UUID AS user_id
FROM cart_items AS ci
INNER JOIN carts AS c ON c.id = ci.cart_id
INNER JOIN users AS u ON u.id = c.owner_id
WHERE ci.product_id = sqlc.arg('product_id')
    AND u.deleted_at IS NULL;
