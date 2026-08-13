-- name: LockCartByID :one
-- ID からカートを 1 件、悲観ロック（FOR UPDATE）して取得する。同一カートへの並行更新を、取得から
-- commit まで直列化する。複数件をロックする場合の順序は呼び出し側が ID 昇順で固定する。
-- 存在しない場合は 0 行（NotFound）。
SELECT sqlc.embed(c)
FROM carts AS c
WHERE c.id = sqlc.arg('id')
FOR UPDATE;
