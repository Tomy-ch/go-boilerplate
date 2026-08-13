
-- === source: database/dml/query_service/purchase/select_purchase_detail_by_id.sql ===
-- name: GetPurchaseDetailForUser :one
-- 認証主体の購入本体 1 件を取得する。ステータス名は購入ステータスマスタとの結合で解決する。
-- 所有権は WHERE 述語（user_id 一致）で担保し、他人・不存在はいずれも 0 行（NotFound で秘匿）。
-- 支払い日時（paid_at）は未支払いなら NULL、キャンセル日時（canceled_at）は未キャンセルなら NULL。
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
    p.paid_at,
    p.canceled_at
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.id = @id AND p.user_id = @user_id;

-- name: ListPurchaseDetailItemsForUser :many
-- 購入明細を products との結合で商品名込みに取得する（集約跨ぎの read 投影）。
-- product_id は FK 制約により products と常に結合可能。id 昇順で安定整列する。
SELECT
    d.product_id,
    pr.name AS product_name,
    d.quantity,
    d.unit_price
FROM purchase_details AS d
INNER JOIN products AS pr ON d.product_id = pr.id
WHERE d.purchase_id = @purchase_id_param
ORDER BY d.id;

-- === source: database/dml/query_service/purchase/select_purchase_summary.sql ===
-- name: SummarizePurchasesByUserID :many
-- 指定ユーザーの購入をステータス単位に集計し、購入ステータスマスタの表示順（sort_key 昇順）で返します。
-- 所有権は user_id の等値条件で閉じるため、他ユーザーの購入は集計に混入しません。
-- 既存の複合インデックス purchases (user_id, ordered_at DESC, id DESC) の先頭列で絞り込みます。
-- キャンセル済み（canceled_at 設定済み）の購入も対象に含めます。キャンセルはステータス別内訳の
-- 1 要素として返るため、除外すると内訳と総計が食い違います。
-- 総件数・合計金額はこの結果行を畳み込んで算出します（単一スナップショットで整合させるため）。
-- ステータス名は購入ステータスマスタとの結合で解決します。
SELECT
    ps.id AS status_id,
    ps.name AS status_name,
    COUNT(p.id)::BIGINT AS purchase_count,
    COALESCE(SUM(p.total_amount), 0)::BIGINT AS total_amount
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.user_id = sqlc.arg('user_id')
GROUP BY ps.id, ps.name, ps.sort_key
ORDER BY ps.sort_key ASC;
