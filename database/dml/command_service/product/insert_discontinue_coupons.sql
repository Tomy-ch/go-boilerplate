-- name: InsertDiscontinueCoupons :execrows
-- 採番済みの id と受給者 user_id を 1 対 1 で zip し、同じ条件のクーポンを一括発行する。
-- 発行枚数は受給者の数で決まり、往復はこの 1 文だけ（件数に比例して増えない）。
-- id をドメイン層で採番するのは ADR-0037 (uuidv7-identifiers) の要請であり、そのため
-- 受給者の取得とこの挿入は 2 文に分かれる。分かれても往復は件数に依存しない。
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
    FROM unnest(sqlc.arg('ids')::UUID[]) WITH ORDINALITY AS i(id, ord)
    INNER JOIN unnest(sqlc.arg('user_ids')::UUID[]) WITH ORDINALITY AS u(user_id, ord)
        ON i.ord = u.ord
) AS ids;
