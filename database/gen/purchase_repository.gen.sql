
-- === source: database/dml/repository/purchase/lock_purchase_by_id.sql ===
-- name: LockPurchaseByID :one
-- ID から購入を 1 件、購入行のみ悲観ロック（FOR UPDATE OF p）して取得する。支払いの状態遷移の
-- 競合（同一購入への並行支払い）を購入行ロックで直列化する（結合先の固定参照マスタはロックしない）。
-- 現在状態は購入ステータスマスタとの結合で code を解決する。存在しない場合は 0 行（NotFound）。
SELECT
    ps.code AS status_code,
    sqlc.embed(p)
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.id = @id
FOR UPDATE OF p;

-- === source: database/dml/repository/purchase/select_purchase_by_id.sql ===
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
-- 支払い日時（paid_at）は未支払いなら NULL、キャンセル日時（canceled_at）は未キャンセルなら NULL。
-- 存在しない場合は 0 行（NotFound）。
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
WHERE p.id = @id;

-- name: ListPurchaseDetailsByPurchaseID :many
-- 購入 ID から明細を id 昇順で取得する。
SELECT sqlc.embed(d)
FROM purchase_details AS d
WHERE d.purchase_id = @purchase_id_param
ORDER BY d.id;

-- === source: database/dml/repository/purchase/select_purchases_feed.sql ===
-- name: ListPurchasesFeedFirst :many
-- 指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で先頭ページ取得する。
-- ステータス名は購入ステータスマスタとの結合で解決する（購入集約に属する固定参照マスタへの
-- 一意な等結合であり、単一集約の read）。一覧は概要のみで明細は含まない。
SELECT
    p.id,
    p.code,
    p.total_amount,
    p.ordered_at,
    ps.id AS status_id,
    ps.name AS status_name
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.user_id = sqlc.arg('user_id')
ORDER BY p.ordered_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');

-- name: ListPurchasesFeedAfter :many
-- (ordered_at DESC, id DESC) の keyset 境界より過去の購入履歴を返す。境界は直前ページ末尾行の
-- (ordered_at, id) で、ordered_at 同値は id で安定にタイブレークする。
SELECT
    p.id,
    p.code,
    p.total_amount,
    p.ordered_at,
    ps.id AS status_id,
    ps.name AS status_name
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.user_id = sqlc.arg('user_id')
    AND (
        p.ordered_at < sqlc.arg('after_ordered_at')
        OR (p.ordered_at = sqlc.arg('after_ordered_at') AND p.id < sqlc.arg('after_id'))
    )
ORDER BY p.ordered_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');

-- === source: database/dml/repository/purchase/update_purchase_paid.sql ===
-- name: UpdatePurchasePaid :exec
-- 購入を支払い済み状態へ更新する。擬似決済のため単一集約（purchases）のみを更新し、在庫操作は伴わない。
-- status_id は code から解決し（seed UUID を焼き込まない）、paid_at はドメインが決定した時刻（引数）を書き込み、
-- イベント payload・レスポンスと同一時刻に揃える。対象行は呼び出し側が FOR UPDATE で取得・検証済みのため、
-- 遷移可否ガードは付けない（ドメインが SoT）。
UPDATE purchases
SET
    status_id = (
        SELECT ps.id FROM purchase_statuses AS ps
        WHERE ps.code = @status_code
    ),
    paid_at = @paid_at,
    updated_at = NOW()
WHERE purchases.id = @id;
