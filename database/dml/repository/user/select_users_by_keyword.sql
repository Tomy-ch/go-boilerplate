-- name: SearchUsers :many
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.search_text ILIKE ANY(sqlc.arg('patterns_param')::TEXT[])
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');

-- name: SearchActiveUsers :many
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.search_text ILIKE ANY(sqlc.arg('patterns_param')::TEXT[])
    AND u.deleted_at IS NULL
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');

-- name: SearchDeletedUsers :many
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.search_text ILIKE ANY(sqlc.arg('patterns_param')::TEXT[])
    AND u.deleted_at IS NOT NULL
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');
