-- name: CountDiscontinueAffectedCarts :one
-- 廃番対象の商品を明細に持つカートの件数を返す。所有者が確定していないゲストのカートも数える。
-- 受給者の抽出（SelectDiscontinueCouponRecipients）と母集団が異なるのは意図で、
-- ゲストのカートは影響を受けるが受給者にならないため 2 つの数は一致しない。
SELECT COUNT(*)
FROM cart_items AS ci
WHERE ci.product_id = sqlc.arg('product_id');
