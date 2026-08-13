-- name: GetCartBySessionToken :one
-- セッショントークンからカートを 1 件取得する。存在しない場合は 0 行（NotFound）。
-- 所有者が確定したカートは session_token を持たない（carts_owner_exclusive）ため、この経路では引けない。
-- 有効期限で絞らない理由は GetCartByOwnerID と同じ。
SELECT sqlc.embed(c)
FROM carts AS c
WHERE c.session_token = sqlc.arg('session_token');
