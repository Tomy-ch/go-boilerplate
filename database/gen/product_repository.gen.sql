
-- === source: database/dml/repository/product/count_product.sql ===
-- name: CountProducts :one
-- 登録済みの商品総数と、そのうち公開済みの商品数を返します。
-- 「公開中」を定義するのは Product.IsPublished で、FILTER 句はその実行形です。片方だけ変更しないこと。
SELECT
    COUNT(*)::BIGINT AS total_count,
    (COUNT(*) FILTER (WHERE published_at IS NOT NULL))::BIGINT AS published_count
FROM products;

-- name: CountPublishedProductsByFilter :one
-- 公開済み商品のうち、商品一覧と同じ検索条件に一致する件数を返します。
SELECT COUNT(*)::BIGINT AS count
FROM products AS p
WHERE p.published_at IS NOT NULL
    AND (sqlc.narg('category_id')::UUID IS NULL OR p.category_id = sqlc.narg('category_id'))
    AND (sqlc.narg('status_id')::UUID IS NULL OR p.status_id = sqlc.narg('status_id'))
    AND (sqlc.narg('min_price')::NUMERIC IS NULL OR p.price >= sqlc.narg('min_price'))
    AND (sqlc.narg('max_price')::NUMERIC IS NULL OR p.price <= sqlc.narg('max_price'))
    AND (sqlc.narg('min_quantity')::INTEGER IS NULL OR p.quantity >= sqlc.narg('min_quantity'))
    AND (sqlc.narg('max_quantity')::INTEGER IS NULL OR p.quantity <= sqlc.narg('max_quantity'))
    AND (
        sqlc.narg('keyword')::TEXT IS NULL
        OR p.name ILIKE '%' || sqlc.narg('keyword') || '%'
        OR p.description ILIKE '%' || sqlc.narg('keyword') || '%'
    );

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
    published_at
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
    sqlc.arg('published_at')
);

-- === source: database/dml/repository/product/insert_product_image.sql ===
-- name: CreateProductImage :exec
-- 商品画像を 1 件登録する。生存行の (product_id, sort_key) は部分 UNIQUE インデックスが一意に保つため、
-- 同一商品の同じ表示順へ二重に登録すると 23505 で失敗する。
INSERT INTO product_images (
    id,
    product_id,
    image_path,
    sort_key
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('product_id'),
    sqlc.arg('image_path'),
    sqlc.arg('sort_key')
);

-- === source: database/dml/repository/product/select_existing_image_paths.sql ===
-- name: ListExistingProductImagePaths :many
-- 与えた画像パスのうち、いずれかの商品が実際に参照しているものを返す。
-- 未参照オブジェクトの回収（product-image-gc）で「消してよいか」を判定する取得元で、
-- ここに現れなかったパスが孤児にあたる。
-- 論理削除された画像は差し替え履歴であって現在の参照ではないため、生存行だけを参照元として数える。
SELECT DISTINCT pi.image_path
FROM product_images AS pi
WHERE pi.image_path = ANY(sqlc.arg('image_paths')::TEXT [])
    AND pi.deleted_at IS NULL;

-- === source: database/dml/repository/product/select_low_stock_products.sql ===
-- name: ListLowStockProducts :many
-- 在庫が警告閾値以下の商品を、在庫の少ない順（同数は ID 昇順）で最大 limit 件取得します。
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
-- 閾値未設定（NULL）の商品は WHERE で明示的に除外する（意味は docs/spec/product/domain.md の
-- Product.FindAllLowStock を参照）。
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
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
-- 公開中のみを返す GetPublishedProductByID とは可視範囲が異なり、未公開商品も返します
-- （用途は docs/spec/product/domain.md の Product.FindByID を参照）。
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
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
SELECT
    ps.name AS status_name,
    pc.name AS category_name,
    sqlc.embed(p)
FROM products AS p
INNER JOIN product_statuses AS ps ON p.status_id = ps.id
INNER JOIN product_categories AS pc ON p.category_id = pc.id
WHERE p.id = sqlc.arg('product_id_param')
FOR UPDATE OF p;

-- === source: database/dml/repository/product/select_product_images.sql ===
-- name: ListProductImagesByProductIDs :many
-- 複数の商品 ID から画像をまとめて取得する。商品 1 件ずつの取得を件数分繰り返さないための一括版で、
-- 並びは商品 ID 昇順・同一商品内は表示順（sort_key）昇順。product_ids が空の場合は 0 行。
-- 生存行だけを返す（論理削除の意味は docs/spec/product/domain.md の Image 節を参照）。
SELECT sqlc.embed(pi)
FROM product_images AS pi
WHERE pi.product_id = ANY(sqlc.arg('product_ids')::UUID [])
    AND pi.deleted_at IS NULL
ORDER BY pi.product_id, pi.sort_key;

-- === source: database/dml/repository/product/select_products.sql ===
-- name: ListPublishedProductsDescFirst :many
-- 公開済み商品を (published_at DESC, id DESC) の安定順で keyset ページネーション取得します。
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
-- category_id / status_id / keyword / price・quantity の上下限は指定時のみ絞り込みます。
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
    AND (sqlc.narg('min_price')::NUMERIC IS NULL OR p.price >= sqlc.narg('min_price'))
    AND (sqlc.narg('max_price')::NUMERIC IS NULL OR p.price <= sqlc.narg('max_price'))
    AND (sqlc.narg('min_quantity')::INTEGER IS NULL OR p.quantity >= sqlc.narg('min_quantity'))
    AND (sqlc.narg('max_quantity')::INTEGER IS NULL OR p.quantity <= sqlc.narg('max_quantity'))
    AND (
        sqlc.narg('keyword')::TEXT IS NULL
        OR p.name ILIKE '%' || sqlc.narg('keyword') || '%'
        OR p.description ILIKE '%' || sqlc.narg('keyword') || '%'
    )
ORDER BY p.published_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');

-- name: ListPublishedProductsDescAfter :many
-- 公開済み商品を (published_at DESC, id DESC) の安定順で keyset ページネーション取得します。
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
-- category_id / status_id / keyword / price・quantity の上下限は指定時のみ絞り込みます。
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
    AND (sqlc.narg('min_price')::NUMERIC IS NULL OR p.price >= sqlc.narg('min_price'))
    AND (sqlc.narg('max_price')::NUMERIC IS NULL OR p.price <= sqlc.narg('max_price'))
    AND (sqlc.narg('min_quantity')::INTEGER IS NULL OR p.quantity >= sqlc.narg('min_quantity'))
    AND (sqlc.narg('max_quantity')::INTEGER IS NULL OR p.quantity <= sqlc.narg('max_quantity'))
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
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
-- category_id / status_id / keyword / price・quantity の上下限は指定時のみ絞り込みます。
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
    AND (sqlc.narg('min_price')::NUMERIC IS NULL OR p.price >= sqlc.narg('min_price'))
    AND (sqlc.narg('max_price')::NUMERIC IS NULL OR p.price <= sqlc.narg('max_price'))
    AND (sqlc.narg('min_quantity')::INTEGER IS NULL OR p.quantity >= sqlc.narg('min_quantity'))
    AND (sqlc.narg('max_quantity')::INTEGER IS NULL OR p.quantity <= sqlc.narg('max_quantity'))
    AND (
        sqlc.narg('keyword')::TEXT IS NULL
        OR p.name ILIKE '%' || sqlc.narg('keyword') || '%'
        OR p.description ILIKE '%' || sqlc.narg('keyword') || '%'
    )
ORDER BY p.published_at ASC, p.id ASC
LIMIT sqlc.arg('limit_param');

-- name: ListPublishedProductsAscAfter :many
-- 公開済み商品を (published_at ASC, id ASC) の安定順で keyset ページネーション取得します。
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
-- category_id / status_id / keyword / price・quantity の上下限は指定時のみ絞り込みます。
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
    AND (sqlc.narg('min_price')::NUMERIC IS NULL OR p.price >= sqlc.narg('min_price'))
    AND (sqlc.narg('max_price')::NUMERIC IS NULL OR p.price <= sqlc.narg('max_price'))
    AND (sqlc.narg('min_quantity')::INTEGER IS NULL OR p.quantity >= sqlc.narg('min_quantity'))
    AND (sqlc.narg('max_quantity')::INTEGER IS NULL OR p.quantity <= sqlc.narg('max_quantity'))
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
-- ロック順序を id 昇順に固定することで、複数商品を同時にロックする処理同士のデッドロックを構造的に避けます（ADR-0034 (ordered-pessimistic-row-locks)）。
-- 不存在の ID は結果に現れないため、返る件数は引数より少なくなり得ます。
-- ロック対象は products のみで、結合する固定参照マスタはロックしません（FOR UPDATE OF p）。
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
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
-- status_name / category_name は固定参照マスタの解決値（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
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

-- === source: database/dml/repository/product/soft_delete_product_images.sql ===
-- name: SoftDeleteProductImages :exec
-- 商品が現在参照している画像をまとめて論理削除する。既に論理削除済みの行は削除日時を上書きしない。
-- 生存行が無い商品に対しては 0 行更新となり、成功として返る。
UPDATE product_images
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE product_images.product_id = sqlc.arg('product_id')
    AND product_images.deleted_at IS NULL;

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
    lock_version = products.lock_version + 1,
    updated_at = NOW()
WHERE products.id = sqlc.arg('id')
    AND products.lock_version = sqlc.arg('current_version')
RETURNING products.lock_version;

-- === source: database/dml/repository/product/update_product_stock.sql ===
-- name: UpdateProductStock :one
-- 在庫数を更新し、採番後のバージョンを返します。
-- lock_version の加算は SQL 側で行う（採番の権威の置き場所は docs/spec/product/domain.md の
-- Product.Update を参照）。
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
