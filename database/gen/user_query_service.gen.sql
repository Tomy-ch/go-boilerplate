
-- === source: database/dml/query_service/user/count_user_by_keyword.sql ===
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

-- === source: database/dml/query_service/user/select_users_by_keyword.sql ===
-- name: SearchUsers :many
SELECT
    p.name AS prefecqture_name,
    sqlc.embed(u)
FROM users AS u
INNER JOIN prefectures AS p ON u.prefecture_id = p.id
WHERE u.search_text ILIKE ANY(sqlc.arg('patterns_param')::TEXT [])
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');

-- name: SearchActiveUsers :many
SELECT
    p.name AS prefecqture_name,
    sqlc.embed(u)
FROM users AS u
INNER JOIN prefectures AS p ON u.prefecture_id = p.id
WHERE u.search_text ILIKE ANY(sqlc.arg('patterns_param')::TEXT [])
    AND u.deleted_at IS NULL
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');

-- name: SearchDeletedUsers :many
SELECT
    p.name AS prefecqture_name,
    sqlc.embed(u)
FROM users AS u
INNER JOIN prefectures AS p ON u.prefecture_id = p.id
WHERE u.search_text ILIKE ANY(sqlc.arg('patterns_param')::TEXT [])
    AND u.deleted_at IS NOT NULL
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');
