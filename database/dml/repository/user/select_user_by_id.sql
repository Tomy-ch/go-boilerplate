-- name: GetUserDomainByID :one
SELECT
    u.id,
    u.first_name,
    u.last_name,
    u.email,
    u.phone,
    u.prefecture_id,
    p.name AS prefecture_name,
    u.city,
    u.street,
    u.building,
    u.postal_code,
    u.deleted_at
FROM users u
JOIN prefectures p ON u.prefecture_id = p.id
WHERE u.id = sqlc.arg(user_id_param);
