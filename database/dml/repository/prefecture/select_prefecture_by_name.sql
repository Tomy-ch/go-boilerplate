-- name: GetPrefectureDomainByName :one
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
WHERE p.name = sqlc.arg('name_param');
