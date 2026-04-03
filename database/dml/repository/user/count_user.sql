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
