
-- === source: database/dml/repository/user/count_user.sql ===
-- name: CountUsersByDeletedState :one
SELECT COUNT(*)
FROM users AS u
WHERE CASE sqlc.arg('deleted_state')::DELETED_STATE
        WHEN 'active' THEN u.deleted_at IS NULL
        WHEN 'deleted' THEN u.deleted_at IS NOT NULL
        ELSE TRUE
    END;

-- === source: database/dml/repository/user/insert_user.sql ===
-- name: CreateUser :exec
INSERT INTO users (
    id,
    first_name,
    last_name,
    password_hash,
    email,
    phone,
    prefecture_id,
    city,
    street,
    building,
    postal_code,
    created_at,
    updated_at
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('first_name'),
    sqlc.arg('last_name'),
    sqlc.arg('password_hash'),
    sqlc.arg('email'),
    sqlc.arg('phone'),
    sqlc.arg('prefecture_id'),
    sqlc.arg('city'),
    sqlc.arg('street'),
    sqlc.arg('building'),
    sqlc.arg('postal_code'),
    sqlc.arg('created_at'),
    sqlc.arg('updated_at')
);

-- === source: database/dml/repository/user/select_user_by_id.sql ===
-- name: GetUserByID :one
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.id = sqlc.arg('user_id_param');

-- === source: database/dml/repository/user/select_users.sql ===
-- name: ListUsers :many
SELECT sqlc.embed(u)
FROM users AS u
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');

-- === source: database/dml/repository/user/select_users_by_keyword.sql ===
-- name: ListUsersByKeywords :many
SELECT sqlc.embed(u)
FROM users AS u
WHERE CASE sqlc.arg('deleted_state')::DELETED_STATE
        WHEN 'active' THEN u.deleted_at IS NULL
        WHEN 'deleted' THEN u.deleted_at IS NOT NULL
        ELSE TRUE
    END
    AND u.search_text ILIKE ALL(sqlc.arg('patterns_param')::TEXT [])
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');
