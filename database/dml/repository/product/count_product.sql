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
        sqlc.narg('keyword')::TEXT IS NULL
        OR p.name ILIKE '%' || sqlc.narg('keyword') || '%'
        OR p.description ILIKE '%' || sqlc.narg('keyword') || '%'
    );
