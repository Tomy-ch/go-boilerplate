-- name: LockCartsByIDs :many
-- ID の集合からカート群を、更新のために悲観ロック（FOR UPDATE）して取得する。
-- id 昇順の ORDER BY を外さないこと。複数件のロックをこの単一文の外へ分割しないこと
-- （ADR-0035 (ordered-pessimistic-row-locks)）。
-- 不存在の ID は結果に現れないため、返る件数は引数より少なくなり得る。
SELECT sqlc.embed(c)
FROM carts AS c
WHERE c.id = ANY(@cart_ids_param::UUID [])
ORDER BY c.id
FOR UPDATE;
