-- name: ListProductImagesByProductIDs :many
-- 複数の商品 ID から画像をまとめて取得する。商品 1 件ずつの取得を件数分繰り返さないための一括版で、
-- 並びは商品 ID 昇順・同一商品内は表示順（sort_key）昇順。product_ids が空の場合は 0 行。
-- 論理削除された画像は差し替え履歴であって現在の画像ではないため、生存行だけを返す。
SELECT sqlc.embed(pi)
FROM product_images AS pi
WHERE pi.product_id = ANY(sqlc.arg('product_ids')::UUID [])
    AND pi.deleted_at IS NULL
ORDER BY pi.product_id, pi.sort_key;
