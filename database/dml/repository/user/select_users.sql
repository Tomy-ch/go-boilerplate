-- name: ListUsers :many
SELECT sqlc.embed(u)
FROM users AS u
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');
