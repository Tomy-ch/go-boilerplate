-- name: ListUsersFeedFirst :many
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.deleted_at IS NULL
ORDER BY u.created_at DESC, u.id DESC
LIMIT sqlc.arg('limit_param');

-- name: ListUsersFeedAfter :many
-- (created_at DESC, id DESC) の keyset 境界より過去の未削除ユーザーを返します。
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.deleted_at IS NULL
    AND (
        u.created_at < sqlc.arg('after_created_at')
        OR (u.created_at = sqlc.arg('after_created_at') AND u.id < sqlc.arg('after_id'))
    )
ORDER BY u.created_at DESC, u.id DESC
LIMIT sqlc.arg('limit_param');
