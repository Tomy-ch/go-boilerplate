-- name: GetPurchaseByID :one
-- ID から購入を 1 件取得する。現在状態は購入ステータスマスタとの結合で code を解決する
-- （status_id は SoT、code は集約が状態機械の判定に用いる業務キー）。存在しない場合は 0 行（NotFound）。
SELECT
    ps.code AS status_code,
    sqlc.embed(p)
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.id = @id;

-- name: GetPurchaseDetailByID :one
-- ID から購入詳細（読み取りモデル）を 1 件取得する。ステータス名は購入ステータスマスタとの結合で
-- 解決済み（購入集約に属する固定参照マスタへの一意な等結合であり、単一集約の read）。
-- キャンセル日時（canceled_at）は未キャンセルなら NULL。存在しない場合は 0 行（NotFound）。
SELECT
    p.id,
    p.code,
    p.user_id,
    ps.id AS status_id,
    ps.name AS status_name,
    p.subtotal_amount,
    p.tax_amount,
    p.shipping_fee,
    p.total_amount,
    p.ordered_at,
    p.canceled_at
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.id = @id;

-- name: ListPurchaseDetailsByPurchaseID :many
-- 購入 ID から明細を id 昇順で取得する。
SELECT sqlc.embed(d)
FROM purchase_details AS d
WHERE d.purchase_id = @purchase_id_param
ORDER BY d.id;
