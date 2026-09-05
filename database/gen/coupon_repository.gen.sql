
-- === source: database/dml/repository/coupon/select_coupon_count_by_scope_target_product_id.sql ===
-- name: SelectCouponCountByScopeTargetProductID :one
-- 指定商品を適用範囲の対象として発行されたクーポンの枚数を返す。
-- 廃番を再実行したときに、新たな発行を伴わずに実績を返す経路が引く。
SELECT COUNT(*)
FROM coupons
WHERE coupons.scope_target_id = sqlc.arg('product_id');
