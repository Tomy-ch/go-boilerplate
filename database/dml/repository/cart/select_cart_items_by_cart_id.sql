-- name: ListCartItemsByCartID :many
-- カート ID から明細を取得する。並びは追加日時の昇順、同時刻は ID 昇順で決着させる。
-- 切り捨ての順序をこの並びに依存させてはならない（その判定は追加日時の値から集約が行う）。
SELECT sqlc.embed(ci)
FROM cart_items AS ci
WHERE ci.cart_id = sqlc.arg('cart_id')
ORDER BY ci.added_at, ci.id;
