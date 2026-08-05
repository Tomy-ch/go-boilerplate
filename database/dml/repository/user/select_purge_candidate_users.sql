-- name: ListPurgeCandidateUserIDsFirst :many
-- 論理削除日時が cutoff より古いユーザーの ID を、ID 昇順の keyset で最大 limit_param 件取得する。
-- 先頭ページを返す。カーソル以降は対の After クエリが担う。
SELECT id
FROM users
WHERE deleted_at IS NOT NULL
    AND deleted_at < sqlc.arg('cutoff')
ORDER BY id ASC
LIMIT sqlc.arg('limit_param');

-- name: ListPurgeCandidateUserIDsAfter :many
-- 論理削除日時が cutoff より古いユーザーの ID を、ID 昇順の keyset で最大 limit_param 件取得する。
-- 物理削除の対象にならない行（購入を持つユーザー）を挟んでも前進できるよう、境界を after_id で受け取る。
-- 先頭ページは対の First クエリが担う。
SELECT id
FROM users
WHERE deleted_at IS NOT NULL
    AND deleted_at < sqlc.arg('cutoff')
    AND id > sqlc.arg('after_id')
ORDER BY id ASC
LIMIT sqlc.arg('limit_param');
