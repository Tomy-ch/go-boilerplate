-- name: ListPublishedProductsDescFirst :many
-- 公開済み商品を (published_at DESC, id DESC) の安定順で keyset ページネーション取得します。
-- 条件と注意点は ListPublishedProductsDescFirst を参照。
-- 本ファイルの他 3 本も、ソート軸とページ方向以外はこの条件と注意点を共有します。
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
        sqlc.narg('category_codes')::SMALLINT[] IS NULL
        OR p.category_id IN (
            SELECT c.id FROM product_categories AS c
            WHERE c.code = ANY(sqlc.narg('category_codes')::SMALLINT[])
        )
    )
    AND (
        sqlc.narg('status_codes')::SMALLINT[] IS NULL
        OR p.status_id IN (
            SELECT s.id FROM product_statuses AS s
            WHERE s.code = ANY(sqlc.narg('status_codes')::SMALLINT[])
        )
    )
    AND (sqlc.narg('min_price')::NUMERIC IS NULL OR p.price >= sqlc.narg('min_price'))
    AND (sqlc.narg('max_price')::NUMERIC IS NULL OR p.price <= sqlc.narg('max_price'))
    AND (sqlc.narg('min_quantity')::INTEGER IS NULL OR p.quantity >= sqlc.narg('min_quantity'))
    AND (sqlc.narg('max_quantity')::INTEGER IS NULL OR p.quantity <= sqlc.narg('max_quantity'))
    AND (
        sqlc.narg('discontinued')::BOOLEAN IS NULL
        OR (p.discontinued_at IS NOT NULL) = sqlc.narg('discontinued')
    )
    AND (
        sqlc.narg('keyword')::TEXT IS NULL
        OR p.name ILIKE '%' || sqlc.narg('keyword') || '%'
        OR p.description ILIKE '%' || sqlc.narg('keyword') || '%'
    )
ORDER BY p.published_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');

-- name: ListPublishedProductsDescAfter :many
-- 公開済み商品を (published_at DESC, id DESC) の安定順で keyset ページネーション取得します。
-- 条件と注意点は ListPublishedProductsDescFirst を参照。
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
        sqlc.narg('category_codes')::SMALLINT[] IS NULL
        OR p.category_id IN (
            SELECT c.id FROM product_categories AS c
            WHERE c.code = ANY(sqlc.narg('category_codes')::SMALLINT[])
        )
    )
    AND (
        sqlc.narg('status_codes')::SMALLINT[] IS NULL
        OR p.status_id IN (
            SELECT s.id FROM product_statuses AS s
            WHERE s.code = ANY(sqlc.narg('status_codes')::SMALLINT[])
        )
    )
    AND (sqlc.narg('min_price')::NUMERIC IS NULL OR p.price >= sqlc.narg('min_price'))
    AND (sqlc.narg('max_price')::NUMERIC IS NULL OR p.price <= sqlc.narg('max_price'))
    AND (sqlc.narg('min_quantity')::INTEGER IS NULL OR p.quantity >= sqlc.narg('min_quantity'))
    AND (sqlc.narg('max_quantity')::INTEGER IS NULL OR p.quantity <= sqlc.narg('max_quantity'))
    AND (
        sqlc.narg('discontinued')::BOOLEAN IS NULL
        OR (p.discontinued_at IS NOT NULL) = sqlc.narg('discontinued')
    )
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
-- 条件と注意点は ListPublishedProductsDescFirst を参照。
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
        sqlc.narg('category_codes')::SMALLINT[] IS NULL
        OR p.category_id IN (
            SELECT c.id FROM product_categories AS c
            WHERE c.code = ANY(sqlc.narg('category_codes')::SMALLINT[])
        )
    )
    AND (
        sqlc.narg('status_codes')::SMALLINT[] IS NULL
        OR p.status_id IN (
            SELECT s.id FROM product_statuses AS s
            WHERE s.code = ANY(sqlc.narg('status_codes')::SMALLINT[])
        )
    )
    AND (sqlc.narg('min_price')::NUMERIC IS NULL OR p.price >= sqlc.narg('min_price'))
    AND (sqlc.narg('max_price')::NUMERIC IS NULL OR p.price <= sqlc.narg('max_price'))
    AND (sqlc.narg('min_quantity')::INTEGER IS NULL OR p.quantity >= sqlc.narg('min_quantity'))
    AND (sqlc.narg('max_quantity')::INTEGER IS NULL OR p.quantity <= sqlc.narg('max_quantity'))
    AND (
        sqlc.narg('discontinued')::BOOLEAN IS NULL
        OR (p.discontinued_at IS NOT NULL) = sqlc.narg('discontinued')
    )
    AND (
        sqlc.narg('keyword')::TEXT IS NULL
        OR p.name ILIKE '%' || sqlc.narg('keyword') || '%'
        OR p.description ILIKE '%' || sqlc.narg('keyword') || '%'
    )
ORDER BY p.published_at ASC, p.id ASC
LIMIT sqlc.arg('limit_param');

-- name: ListPublishedProductsAscAfter :many
-- 公開済み商品を (published_at ASC, id ASC) の安定順で keyset ページネーション取得します。
-- 条件と注意点は ListPublishedProductsDescFirst を参照。
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
        sqlc.narg('category_codes')::SMALLINT[] IS NULL
        OR p.category_id IN (
            SELECT c.id FROM product_categories AS c
            WHERE c.code = ANY(sqlc.narg('category_codes')::SMALLINT[])
        )
    )
    AND (
        sqlc.narg('status_codes')::SMALLINT[] IS NULL
        OR p.status_id IN (
            SELECT s.id FROM product_statuses AS s
            WHERE s.code = ANY(sqlc.narg('status_codes')::SMALLINT[])
        )
    )
    AND (sqlc.narg('min_price')::NUMERIC IS NULL OR p.price >= sqlc.narg('min_price'))
    AND (sqlc.narg('max_price')::NUMERIC IS NULL OR p.price <= sqlc.narg('max_price'))
    AND (sqlc.narg('min_quantity')::INTEGER IS NULL OR p.quantity >= sqlc.narg('min_quantity'))
    AND (sqlc.narg('max_quantity')::INTEGER IS NULL OR p.quantity <= sqlc.narg('max_quantity'))
    AND (
        sqlc.narg('discontinued')::BOOLEAN IS NULL
        OR (p.discontinued_at IS NOT NULL) = sqlc.narg('discontinued')
    )
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
