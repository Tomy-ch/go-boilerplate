-- name: CountSearchUsers :one
SELECT COUNT(*)
FROM users AS u
WHERE u.search_text ILIKE ANY(sqlc.arg('patterns_param')::TEXT []);

-- name: CountSearchActiveUsers :one
SELECT COUNT(*)
FROM users AS u
WHERE u.search_text ILIKE ANY(sqlc.arg('patterns_param')::TEXT [])
    AND u.deleted_at IS NULL;

-- name: CountSearchDeletedUsers :one
SELECT COUNT(*)
FROM users AS u
WHERE u.search_text ILIKE ANY(sqlc.arg('patterns_param')::TEXT [])
    AND u.deleted_at IS NOT NULL;
