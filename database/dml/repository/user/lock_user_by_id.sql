-- name: LockUserByID :one
-- ID から未削除のユーザーを 1 件、悲観ロック（FOR UPDATE）して取得する。
-- 退会は、この排他ロックを進行中購入の判定より前に取ることで購入作成（FOR SHARE）と直列化する。
-- 論理削除済み・不存在はいずれも 0 行（NotFound）。
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.id = sqlc.arg('user_id_param')
    AND u.deleted_at IS NULL
FOR UPDATE;
