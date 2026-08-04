-- name: ListPurgeCandidateUserIDs :many
-- 論理削除日時が cutoff より古いユーザーの ID を、ID 昇順の keyset で最大 limit_param 件取得する。
-- 物理削除の対象にならない行（購入を持つユーザー）を挟んでも前進できるよう、境界を after_id で受け取る。
SELECT id
FROM users
WHERE deleted_at IS NOT NULL
    AND deleted_at < sqlc.arg('cutoff')
    AND (sqlc.narg('after_id')::UUID IS NULL OR id > sqlc.narg('after_id'))
ORDER BY id ASC
LIMIT sqlc.arg('limit_param');
