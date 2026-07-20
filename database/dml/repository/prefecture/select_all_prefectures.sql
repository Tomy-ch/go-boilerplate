-- name: GetPrefectureDomainAll :many
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
ORDER BY p.code ASC;
