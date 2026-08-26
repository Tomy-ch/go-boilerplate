-- name: ListPurgeCandidateUserIDsFirst :many
-- 論理削除日時が cutoff より古いユーザーの ID を、ID 昇順の keyset で最大 limit_param 件取得する（先頭ページ）。
SELECT id
FROM users
WHERE deleted_at IS NOT NULL
    AND deleted_at < sqlc.arg('cutoff')
ORDER BY id ASC
LIMIT sqlc.arg('limit_param');

-- name: ListPurgeCandidateUserIDsAfter :many
-- 論理削除日時が cutoff より古いユーザーの ID を、ID 昇順の keyset で最大 limit_param 件取得する（after_id 以降）。
-- 境界を offset でなく ID で受け取る理由は docs/spec/user/domain.md の FindDeletedBefore を参照。
SELECT id
FROM users
WHERE deleted_at IS NOT NULL
    AND deleted_at < sqlc.arg('cutoff')
    AND id > sqlc.arg('after_id')
ORDER BY id ASC
LIMIT sqlc.arg('limit_param');
