
-- === source: database/dml/repository/purchase/select_purchase_by_id.sql ===
-- name: GetPurchaseByID :one
-- ID から購入を 1 件取得する。存在しない場合は 0 行（NotFound）。
SELECT sqlc.embed(p)
FROM purchases AS p
WHERE p.id = @id;

-- name: ListPurchaseDetailsByPurchaseID :many
-- 購入 ID から明細を id 昇順で取得する。
SELECT sqlc.embed(d)
FROM purchase_details AS d
WHERE d.purchase_id = @purchase_id_param
ORDER BY d.id;
