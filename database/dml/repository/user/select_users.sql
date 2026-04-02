-- name: ListUsers :many
SELECT sqlc.embed(u)
FROM users AS u
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');

-- name: ListActiveUsers :many
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.deleted_at IS NULL
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');

-- name: ListDeletedUsers :many
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.deleted_at IS NOT NULL
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');
