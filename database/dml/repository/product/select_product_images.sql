-- name: ListProductImagesByProductIDs :many
-- 複数の商品 ID から画像をまとめて取得する。商品 1 件ずつの取得を件数分繰り返さないための一括版で、
-- 並びは商品 ID 昇順・同一商品内は表示順（display_sort）昇順。product_ids が空の場合は 0 行。
-- 生存行だけを返す（論理削除の意味は docs/spec/product/domain.md の Image 節を参照）。
SELECT sqlc.embed(pi)
FROM product_images AS pi
WHERE pi.product_id = ANY(sqlc.arg('product_ids')::UUID [])
    AND pi.deleted_at IS NULL
ORDER BY pi.product_id, pi.display_sort;
