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
