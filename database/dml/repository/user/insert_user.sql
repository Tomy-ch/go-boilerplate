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
