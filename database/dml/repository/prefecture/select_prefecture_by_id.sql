-- name: GetPrefectureDomainByID :one
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
WHERE p.id = sqlc.arg('id_param');
