
-- === source: database/dml/command_service/purchase/decrement_product_stock.sql ===
-- name: DecrementProductStock :execrows
-- 在庫を数量分減算する。防御的に quantity >= 減算数を併用し、売り越しをアトミックに弾く（更新 0 行なら在庫不足）。
-- ロック取得後に検証済みのため通常は 0 行にならないが、fail-closed の二重防御として残す（ADR-0100）。
UPDATE products
SET
    quantity = quantity - @quantity_param,
    updated_at = NOW()
WHERE id = @product_id_param
    AND quantity >= @quantity_param;

-- === source: database/dml/command_service/purchase/insert_purchase.sql ===
-- name: InsertPurchase :exec
-- 購入を 1 行 INSERT する。status_id は code から解決する（seed UUID をアプリに焼き込まない）。
-- ordered_at / created_at / updated_at は DB 既定（NOW()）に委ねる。
INSERT INTO purchases (
    id,
    code,
    user_id,
    status_id,
    subtotal_amount,
    tax_amount,
    shipping_fee,
    total_amount
) VALUES (
    @id,
    @code,
    @user_id,
    (
        SELECT ps.id FROM purchase_statuses AS ps
        WHERE ps.code = @status_code
    ),
    @subtotal_amount,
    @tax_amount,
    @shipping_fee,
    @total_amount
);

-- === source: database/dml/command_service/purchase/insert_purchase_detail.sql ===
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

-- === source: database/dml/command_service/purchase/lock_products_for_update.sql ===
-- name: LockProductsForUpdate :many
-- 指定商品を ID 昇順に悲観ロック（FOR UPDATE）し、価格・在庫を返す。
-- ロック順序を id 昇順に固定することで複数商品購入同士のデッドロックを構造的に避ける（ADR-0100）。
SELECT
    p.id,
    p.price,
    p.quantity
FROM products AS p
WHERE p.id = ANY(@product_ids::UUID [])
ORDER BY p.id
FOR UPDATE;
