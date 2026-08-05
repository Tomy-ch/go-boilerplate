-- name: LockActiveUserShareByID :one
-- ID の未削除ユーザーの存在を、共有ロック（FOR SHARE）を取りながら確認する。
-- 共有ロック同士は両立するため同一ユーザーの並行購入は直列化されず、退会が取る FOR UPDATE とだけ
-- 衝突する。これにより「退会の判定通過 → 購入の成立 → 退会の確定」の順序が成立しなくなる。
-- 論理削除済み・不存在はいずれも 0 行（NotFound）。
SELECT u.id
FROM users AS u
WHERE u.id = sqlc.arg('user_id_param')
    AND u.deleted_at IS NULL
FOR SHARE;
