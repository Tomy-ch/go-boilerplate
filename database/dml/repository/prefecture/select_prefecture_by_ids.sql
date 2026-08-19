-- name: GetPrefectureDomainByIDs :many
-- 指定 ID の都道府県をマスタの表示順（sort_key 昇順）で返す。並び順の出所は code ではない。
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
WHERE p.id = ANY(@ids_param::UUID[])
ORDER BY p.sort_key;
