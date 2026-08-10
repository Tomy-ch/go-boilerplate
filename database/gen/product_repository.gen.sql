
-- === source: database/dml/repository/product/count_product.sql ===
-- name: CountProducts :one
-- 登録済みの商品総数と、そのうち公開済みの商品数を返します。
-- 「公開中」を定義するのは Product.IsPublished で、FILTER 句はその実行形です。片方だけ変更しないこと。
SELECT
    COUNT(*)::BIGINT AS total_count,
    (COUNT(*) FILTER (WHERE published_at IS NOT NULL))::BIGINT AS published_count
FROM products;

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

-- === source: database/dml/repository/product/select_existing_image_paths.sql ===
-- name: ListExistingProductImagePaths :many
-- 与えた画像パスのうち、いずれかの商品が実際に参照しているものを返す。
-- 未参照オブジェクトの回収（product-image-gc）で「消してよいか」を判定する取得元で、
-- ここに現れなかったパスが孤児にあたる。商品は論理削除を持たないため、生存行だけが参照元になる。
SELECT DISTINCT image_path
FROM products
WHERE image_path = ANY(sqlc.arg('image_paths')::TEXT []);

-- === source: database/dml/repository/product/select_low_stock_products.sql ===
-- name: ListLowStockProducts :many
-- 在庫が警告閾値以下の商品を、在庫の少ない順（同数は ID 昇順）で最大 limit 件取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- stock_warning_threshold が NULL（閾値未設定）の商品は警告対象外として明示的に除外します。
-- 「在庫僅少」を定義するのは Product.IsLowStock で、以下の条件はその実行形です。片方だけ変更しないこと。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.stock_warning_threshold IS NOT NULL
    AND p.quantity <= p.stock_warning_threshold
ORDER BY p.quantity ASC, p.id ASC
LIMIT sqlc.arg('limit_param');

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
-- name: ListPublishedProductsDescFirst :many
-- 公開済み商品を (published_at DESC, id DESC) の安定順で keyset ページネーション取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- category_id / status_id / keyword は指定時のみ絞り込みます。
-- 「公開中」を定義するのは Product.IsPublished で、published_at の条件はその実行形です。片方だけ変更しないこと。
-- 先頭ページを返します。カーソル以降は対の After クエリが担います。
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
ORDER BY p.published_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');

-- name: ListPublishedProductsDescAfter :many
-- 公開済み商品を (published_at DESC, id DESC) の安定順で keyset ページネーション取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- category_id / status_id / keyword は指定時のみ絞り込みます。
-- 「公開中」を定義するのは Product.IsPublished で、published_at の条件はその実行形です。片方だけ変更しないこと。
-- カーソル以降のページを返します。先頭ページは対の First クエリが担います。
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
        p.published_at < sqlc.arg('after_published_at')
        OR (p.published_at = sqlc.arg('after_published_at') AND p.id < sqlc.arg('after_id'))
    )
ORDER BY p.published_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');

-- name: ListPublishedProductsAscFirst :many
-- 公開済み商品を (published_at ASC, id ASC) の安定順で keyset ページネーション取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- category_id / status_id / keyword は指定時のみ絞り込みます。
-- 「公開中」を定義するのは Product.IsPublished で、published_at の条件はその実行形です。片方だけ変更しないこと。
-- 先頭ページを返します。カーソル以降は対の After クエリが担います。
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
ORDER BY p.published_at ASC, p.id ASC
LIMIT sqlc.arg('limit_param');

-- name: ListPublishedProductsAscAfter :many
-- 公開済み商品を (published_at ASC, id ASC) の安定順で keyset ページネーション取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- category_id / status_id / keyword は指定時のみ絞り込みます。
-- 「公開中」を定義するのは Product.IsPublished で、published_at の条件はその実行形です。片方だけ変更しないこと。
-- カーソル以降のページを返します。先頭ページは対の First クエリが担います。
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
        p.published_at > sqlc.arg('after_published_at')
        OR (p.published_at = sqlc.arg('after_published_at') AND p.id > sqlc.arg('after_id'))
    )
ORDER BY p.published_at ASC, p.id ASC
LIMIT sqlc.arg('limit_param');

-- === source: database/dml/repository/product/select_products_by_ids_for_update.sql ===
-- name: ListProductsByIDsForUpdate :many
-- ID の集合から公開状態を問わない商品群を、更新のために悲観ロック（FOR UPDATE）して取得します。
-- ロック順序を id 昇順に固定することで、複数商品を同時にロックする処理同士のデッドロックを構造的に避けます（ADR-0033 (ordered-pessimistic-row-locks)）。
-- 不存在の ID は結果に現れないため、返る件数は引数より少なくなり得ます。
-- ロック対象は products のみで、結合する固定参照マスタはロックしません（FOR UPDATE OF p）。
-- status_name / category_name は商品の付随表示値。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.id = ANY(@product_ids_param::UUID [])
ORDER BY p.id
FOR UPDATE OF p;

-- === source: database/dml/repository/product/select_published_product_by_id.sql ===
-- name: GetPublishedProductByID :one
-- ID から公開中の単一商品を取得します。
-- status_name / category_name は商品の付随表示値。
-- 固定参照マスタのみを結合し、集約境界をまたがない単一集約 Repository read です。
-- 「公開中」を定義するのは Product.IsPublished で、以下の条件はその実行形です。片方だけ変更しないこと。
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
