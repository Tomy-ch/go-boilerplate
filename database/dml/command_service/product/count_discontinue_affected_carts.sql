-- name: CountDiscontinueAffectedCarts :one
-- 廃番対象の商品を明細に持つカートの件数を返す。所有者が確定していないゲストのカートも数える
-- （受給者との母集団差は docs/spec/usecase/product.md の Workflow — DiscontinueProduct を参照）。
SELECT COUNT(*)
FROM cart_items AS ci
WHERE ci.product_id = sqlc.arg('product_id');
