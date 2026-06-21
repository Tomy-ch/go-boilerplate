
-- === source: database/dml/repository/prefecture/select_prefecture_by_id.sql ===
-- name: GetPrefectureDomainByID :one
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
WHERE p.id = sqlc.arg('id_param');

-- === source: database/dml/repository/prefecture/select_prefecture_by_ids.sql ===
-- name: GetPrefectureDomainByIDs :many
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
WHERE p.id = ANY(@ids_param::UUID [])
ORDER BY p.code;

-- === source: database/dml/repository/prefecture/select_prefecture_by_name.sql ===
-- name: GetPrefectureDomainByName :one
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
WHERE p.name = sqlc.arg('name_param');
