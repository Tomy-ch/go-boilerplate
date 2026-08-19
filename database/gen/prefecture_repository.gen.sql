
-- === source: database/dml/repository/prefecture/select_all_prefectures.sql ===
-- name: GetPrefectureDomainAll :many
-- 全都道府県をマスタの表示順（sort_key 昇順）で返す。code は外部が行を指す静的な別名であり、
-- 並び順の出所ではない。
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
ORDER BY p.sort_key ASC;

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
-- 指定 ID の都道府県をマスタの表示順（sort_key 昇順）で返す。並び順の出所は code ではない。
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
WHERE p.id = ANY(@ids_param::UUID[])
ORDER BY p.sort_key;

-- === source: database/dml/repository/prefecture/select_prefecture_by_name.sql ===
-- name: GetPrefectureDomainByName :one
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
WHERE p.name = sqlc.arg('name_param');
