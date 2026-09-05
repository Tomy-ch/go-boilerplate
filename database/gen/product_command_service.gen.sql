
-- === source: database/dml/command_service/product/count_discontinue_affected_carts.sql ===
-- name: CountDiscontinueAffectedCarts :one
-- 廃番対象の商品を明細に持つカートの件数を返す。所有者が確定していないゲストのカートも数える
-- （受給者との母集団差は docs/spec/usecase/product.md の Workflow — DiscontinueProduct を参照）。
SELECT COUNT(*)
FROM cart_items AS ci
WHERE ci.product_id = sqlc.arg('product_id');

-- === source: database/dml/command_service/product/insert_discontinue_coupons.sql ===
-- name: InsertDiscontinueCoupons :execrows
-- 採番済みの id と受給者 user_id を 1 対 1 で zip し、同じ条件のクーポンを一括発行する
-- （2 文に分かれる理由と往復コストは ADR-0034 の Worked instances を参照）。
-- 2 つの配列は WITH ORDINALITY の行番号で突き合わせる（sqlc が 2 引数形の unnest を解決できない）。
-- 長さが食い違うと内部結合で余った側が落ちるため、呼び出し側が必ず同じ長さで渡す。
INSERT INTO coupons (
    id,
    user_id,
    discount_kind,
    discount_value,
    scope_kind,
    scope_target_id,
    expires_at,
    issued_at
)
SELECT
    ids.id,
    ids.user_id,
    sqlc.arg('discount_kind'),
    sqlc.arg('discount_value'),
    sqlc.arg('scope_kind'),
    sqlc.arg('scope_target_id'),
    sqlc.arg('expires_at'),
    sqlc.arg('issued_at')
FROM (
    SELECT
        i.id,
        u.user_id
    FROM UNNEST(sqlc.arg('ids')::UUID[]) WITH ORDINALITY AS i (id, ord)
    INNER JOIN UNNEST(sqlc.arg('user_ids')::UUID[]) WITH ORDINALITY AS u (user_id, ord)
        ON i.ord = u.ord
) AS ids;

-- === source: database/dml/command_service/product/select_discontinue_coupon_recipients.sql ===
-- name: SelectDiscontinueCouponRecipients :many
-- 廃番対象の商品を明細に持つカートのうち、所有者が確定していて退会もしていないユーザーを重複なく返す。
-- 絞り込みの理由と母集団が確定する時点は docs/spec/usecase/product.md の
-- Workflow — DiscontinueProduct の invariants を参照。
SELECT DISTINCT c.user_id::UUID AS user_id
FROM cart_items AS ci
INNER JOIN carts AS c ON ci.cart_id = c.id
INNER JOIN users AS u ON c.user_id = u.id
WHERE ci.product_id = sqlc.arg('product_id')
    AND u.deleted_at IS NULL;
