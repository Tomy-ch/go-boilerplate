-- name: GetPurchaseDetailForUser :one
-- 認証主体の購入本体 1 件を購入コードで取得する。
-- 所有権は WHERE 述語（user_id 一致）で担保し、他人・不存在はいずれも 0 行（NotFound で秘匿）。
-- 支払い日時（paid_at）は未支払いなら NULL、キャンセル日時（canceled_at）は未キャンセルなら NULL。
-- 適用したクーポンの 2 軸（値引き・適用範囲）は結合で解決する。控えへ写さないのは productName と
-- 同じ線で、発行済みクーポンを書き換える口が無いため結合でも内容がぶれない
-- （docs/spec/usecase/purchase.md の GET 詳細）。未適用なら結合先が無く NULL。
SELECT
    p.id,
    p.code,
    p.user_id,
    ps.id AS status_id,
    ps.code AS status_code,
    ps.name AS status_name,
    p.subtotal_amount,
    p.tax_amount,
    p.shipping_fee,
    p.total_amount,
    p.ordered_at,
    p.paid_at,
    p.canceled_at,
    p.discount_amount,
    c.id AS coupon_id,
    c.discount_kind AS coupon_discount_kind,
    c.discount_value AS coupon_discount_value,
    c.scope_kind AS coupon_scope_kind,
    c.scope_target_id AS coupon_scope_target_id
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
LEFT JOIN coupons AS c ON p.coupon_id = c.id
WHERE p.code = @code AND p.user_id = @user_id;

-- name: ListPurchaseDetailItemsForUser :many
-- 購入明細を products との結合で商品名込みに取得する（集約跨ぎの read 投影）。
-- 本体行から得た購入 ID で引くため、所有権は本体クエリ側で既に閉じている。
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
