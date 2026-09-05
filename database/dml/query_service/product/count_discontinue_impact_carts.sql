-- name: CountDiscontinueImpactCarts :one
-- 廃番の影響を受けるカートの件数を返す。ゲストのカートも数える。
-- 実行側の CountDiscontinueAffectedCarts と同じ条件を持つ。片方だけを変えてはならない
-- （見積もりと実行が食い違うと、押す前に見せた数字の意味が失われる）。
-- 行はロックしないため、返した値は返した瞬間から古くなる。
SELECT COUNT(*)
FROM cart_items AS ci
WHERE ci.product_id = sqlc.arg('product_id');
