
-- === source: database/dml/repository/product/insert_product.sql ===
-- name: CreateProduct :exec
INSERT INTO products (
    id,
    name,
    description,
    price,
    quantity,
    stock_warning_threshold,
    status_id,
    category_id,
    published_at,
    image_path
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('description'),
    sqlc.arg('price'),
    sqlc.arg('quantity'),
    sqlc.arg('stock_warning_threshold'),
    sqlc.arg('status_id'),
    sqlc.arg('category_id'),
    sqlc.arg('published_at'),
    sqlc.arg('image_path')
);

-- === source: database/dml/repository/product/select_product_by_id.sql ===
-- name: GetProductByID :one
-- ID から公開状態を問わない単一商品を取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- 公開中のみを返す GetPublishedProductByID とは可視範囲が異なり、未公開商品も返します
-- （公開日時の設定そのものを更新対象とする管理用途の read-modify-write に用います）。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.id = sqlc.arg('product_id_param');

-- === source: database/dml/repository/product/select_product_by_id_for_update.sql ===
-- name: GetProductByIDForUpdate :one
-- ID から公開状態を問わない単一商品を、更新のために悲観ロック（FOR UPDATE）して取得します。
-- 同一商品への並行書き込み（購入の在庫減算・在庫補充）を行ロックで直列化します。
-- ロック対象は products のみで、結合する固定参照マスタはロックしません（FOR UPDATE OF p）。
-- status_name / category_name は商品の付随表示値。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.id = sqlc.arg('product_id_param')
FOR UPDATE OF p;

-- === source: database/dml/repository/product/select_products.sql ===
-- name: ListPublishedProductsDesc :many
-- 公開済み商品を (published_at DESC, id DESC) の安定順で keyset ページネーション取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- category_id / status_id / keyword は指定時のみ絞り込み、has_after=true の場合は keyset 境界(after_*)より過去へ絞り込みます。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.published_at IS NOT NULL
    AND (sqlc.narg('category_id')::UUID IS NULL OR p.category_id = sqlc.narg('category_id'))
    AND (sqlc.narg('status_id')::UUID IS NULL OR p.status_id = sqlc.narg('status_id'))
    AND (
        sqlc.narg('keyword')::TEXT IS NULL
        OR p.name ILIKE '%' || sqlc.narg('keyword') || '%'
        OR p.description ILIKE '%' || sqlc.narg('keyword') || '%'
    )
    AND (
        NOT sqlc.arg('has_after')::BOOLEAN
        OR p.published_at < sqlc.narg('after_published_at')
        OR (p.published_at = sqlc.narg('after_published_at') AND p.id < sqlc.narg('after_id'))
    )
ORDER BY p.published_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');

-- name: ListPublishedProductsAsc :many
-- 公開済み商品を (published_at ASC, id ASC) の安定順で keyset ページネーション取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- category_id / status_id / keyword は指定時のみ絞り込み、has_after=true の場合は keyset 境界(after_*)より未来へ絞り込みます。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.published_at IS NOT NULL
    AND (sqlc.narg('category_id')::UUID IS NULL OR p.category_id = sqlc.narg('category_id'))
    AND (sqlc.narg('status_id')::UUID IS NULL OR p.status_id = sqlc.narg('status_id'))
    AND (
        sqlc.narg('keyword')::TEXT IS NULL
        OR p.name ILIKE '%' || sqlc.narg('keyword') || '%'
        OR p.description ILIKE '%' || sqlc.narg('keyword') || '%'
    )
    AND (
        NOT sqlc.arg('has_after')::BOOLEAN
        OR p.published_at > sqlc.narg('after_published_at')
        OR (p.published_at = sqlc.narg('after_published_at') AND p.id > sqlc.narg('after_id'))
    )
ORDER BY p.published_at ASC, p.id ASC
LIMIT sqlc.arg('limit_param');

-- === source: database/dml/repository/product/select_published_product_by_id.sql ===
-- name: GetPublishedProductByID :one
-- ID から公開中の単一商品を取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- 公開範囲の定義は一覧取得（ListPublishedProducts*）と同一述語（published_at 非 NULL）で、
-- 非公開・未存在はいずれも該当行なし（0 行）で返ります。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.id = sqlc.arg('product_id_param')
    AND p.published_at IS NOT NULL;

-- === source: database/dml/repository/product/update_product.sql ===
-- name: UpdateProduct :one
-- 楽観ロック条件付きで商品を更新し、採番後のバージョンを返します。
-- lock_version の加算は DB が行い、採番の権威を単一箇所に置きます。
-- WHERE の lock_version 一致が並行更新による lost update を防ぐ本体で、読み込み後に他トランザクションが
-- 更新していた場合は該当行なし（0 行）で返り、呼び出し側が衝突として扱います。
UPDATE products
SET
    name = sqlc.arg('name'),
    description = sqlc.arg('description'),
    price = sqlc.arg('price'),
    quantity = sqlc.arg('quantity'),
    stock_warning_threshold = sqlc.arg('stock_warning_threshold'),
    status_id = sqlc.arg('status_id'),
    category_id = sqlc.arg('category_id'),
    published_at = sqlc.arg('published_at'),
    image_path = sqlc.arg('image_path'),
    lock_version = products.lock_version + 1,
    updated_at = NOW()
WHERE products.id = sqlc.arg('id')
    AND products.lock_version = sqlc.arg('current_version')
RETURNING products.lock_version;

-- === source: database/dml/repository/product/update_product_stock.sql ===
-- name: UpdateProductStock :one
-- 在庫数を更新し、採番後のバージョンを返します。
-- lock_version の加算は DB が行い、採番の権威を単一箇所に置きます。
-- 在庫更新でもバージョンを進めることで、更新前のバージョンを条件とする部分更新（UpdateProduct）が
-- 在庫の変化を上書きせずに 0 行で弾かれます。
-- WHERE の lock_version 一致は、行ロックを取らずに呼ばれた場合に備える二重防御で、
-- 該当行なし（0 行）は呼び出し側が衝突として扱います。
UPDATE products
SET
    quantity = sqlc.arg('quantity'),
    lock_version = products.lock_version + 1,
    updated_at = NOW()
WHERE products.id = sqlc.arg('id')
    AND products.lock_version = sqlc.arg('current_version')
RETURNING products.lock_version;
