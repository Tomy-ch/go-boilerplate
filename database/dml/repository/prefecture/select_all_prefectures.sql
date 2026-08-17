-- name: GetPrefectureDomainAll :many
-- 全都道府県をマスタの表示順（sort_key 昇順）で返す。code は外部が行を指す静的な別名であり、
-- 並び順の出所ではない。順序を変えるときに動かすのは sort_key の側。
SELECT
    p.id,
    p.name,
    p.code
FROM prefectures AS p
ORDER BY p.sort_key ASC;
