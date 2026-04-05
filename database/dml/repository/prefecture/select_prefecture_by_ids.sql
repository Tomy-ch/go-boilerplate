-- name: GetPrefectureDomainByIDs :many
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
WHERE p.id = ANY(@ids_param::UUID [])
ORDER BY p.code;
