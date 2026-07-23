-- name: InsertPurchaseDetail :exec
-- 購入明細を 1 行 INSERT する。unit_price は購入時点の単価スナップショット（USD セント整数）。
INSERT INTO purchase_details (
    id,
    purchase_id,
    product_id,
    quantity,
    unit_price
) VALUES (
    @id,
    @purchase_id,
    @product_id,
    @quantity,
    @unit_price
);
