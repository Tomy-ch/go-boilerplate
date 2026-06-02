
-- === source: database/dml/repository/user/count_user.sql ===
-- name: CountUsers :one
SELECT COUNT(*)
FROM users;

-- name: CountActiveUsers :one
SELECT COUNT(*)
FROM users AS u
WHERE u.deleted_at IS NULL;

-- name: CountDeletedUsers :one
SELECT COUNT(*)
FROM users AS u
WHERE u.deleted_at IS NOT NULL;

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

-- === source: database/dml/repository/user/update_user.sql ===
-- name: UpdateUser :execrows
UPDATE users
SET
    first_name = sqlc.arg('first_name'),
    last_name = sqlc.arg('last_name'),
    password_hash = sqlc.arg('password_hash'),
    email = sqlc.arg('email'),
    phone = sqlc.arg('phone'),
    prefecture_id = sqlc.arg('prefecture_id'),
    city = sqlc.arg('city'),
    street = sqlc.arg('street'),
    building = sqlc.arg('building'),
    postal_code = sqlc.arg('postal_code'),
    updated_at = sqlc.arg('updated_at'),
    deleted_at = sqlc.arg('deleted_at')
WHERE id = sqlc.arg('id');
