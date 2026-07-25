
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

-- === source: database/dml/command_service/purchase/increment_product_stock.sql ===
-- name: IncrementProductStock :execrows
-- 在庫を数量分復元（加算）する。相対更新（quantity + 数量）のため売り越しを生まず在庫不足ガードは不要
-- （購入行ロック下で実行）。対象行が不存在の場合は影響 0 行として呼び出し側で NotFound へ fail-closed 検出する。
UPDATE products
SET
    quantity = quantity + @quantity_param,
    updated_at = NOW()
WHERE id = @product_id_param;

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

-- === source: database/dml/command_service/purchase/lock_purchase_for_update.sql ===
-- name: GetPurchaseByIDForUpdate :one
-- ID から購入を 1 件、購入行のみ悲観ロック（FOR UPDATE OF p）して取得する。キャンセルの状態遷移の
-- 競合（同一購入への並行キャンセル）を購入行ロックで直列化する（結合先の固定参照マスタはロックしない）。
-- 現在状態は購入ステータスマスタとの結合で code を解決する。存在しない場合は 0 行（NotFound）。
SELECT
    ps.code AS status_code,
    sqlc.embed(p)
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.id = @id
FOR UPDATE OF p;

-- === source: database/dml/command_service/purchase/update_purchase_canceled.sql ===
-- name: UpdatePurchaseCanceled :exec
-- 購入をキャンセル状態へ更新する。status_id は code から解決し（seed UUID を焼き込まない）、
-- canceled_at はドメインが決定した時刻（引数）を書き込み、イベント payload・レスポンスと同一時刻に揃える。
-- 対象行は呼び出し側が FOR UPDATE で取得・検証済みのため、遷移可否ガードは付けない（ドメインが SoT）。
UPDATE purchases
SET
    status_id = (
        SELECT ps.id FROM purchase_statuses AS ps
        WHERE ps.code = @status_code
    ),
    canceled_at = @canceled_at,
    updated_at = NOW()
WHERE purchases.id = @id;
