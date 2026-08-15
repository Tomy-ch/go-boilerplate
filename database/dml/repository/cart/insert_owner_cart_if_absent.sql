-- name: CreateOwnerCartIfAbsent :one
-- 所有者のカートが無ければ空のカートを 1 件登録し、既にあればその行をそのまま返す。
-- 一意インデックス（carts_user_id_unique）が単一文の中で裁定するため、同一ユーザーへの並行した
-- 作成が競合しても一意制約違反を上げない。存在確認と作成を分けると、その間に他の要求が作った場合に
-- 23505 でトランザクションごと中断してしまい、同じトランザクションの中では続けられなくなる。
-- 衝突時に user_id を同じ値で書き戻すのは、DO NOTHING では RETURNING が行を返さないため。
INSERT INTO carts (
    id,
    user_id,
    session_token,
    expires_at
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('user_id'),
    NULL,
    sqlc.arg('expires_at')
)
ON CONFLICT ON CONSTRAINT carts_user_id_unique DO UPDATE
    SET
        user_id = excluded.user_id
RETURNING sqlc.embed(carts);
