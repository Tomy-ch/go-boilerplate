-- name: LockUserByID :one
-- ID から未削除のユーザーを 1 件、悲観ロック（FOR UPDATE）して取得する。
-- 論理削除済み・不存在はいずれも 0 行（NotFound）。
-- 取得位置の不変条件は docs/spec/user/usecase.md の DeleteUser を参照（ADR-0035 (ordered-pessimistic-row-locks)）。
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.id = sqlc.arg('user_id_param')
    AND u.deleted_at IS NULL
FOR UPDATE;
