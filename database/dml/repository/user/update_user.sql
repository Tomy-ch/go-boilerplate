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
