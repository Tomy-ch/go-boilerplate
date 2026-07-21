-- name: ListPublishedProductsDesc :many
-- 公開済み商品を (published_at DESC, id DESC) の安定順で keyset ページネーション取得します。
-- category_id / status_id / keyword は指定時のみ絞り込み、has_after=true の場合は keyset 境界(after_*)より過去へ絞り込みます。
SELECT sqlc.embed(p)
FROM products AS p
WHERE p.published_at IS NOT NULL
    AND (sqlc.narg('category_id')::uuid IS NULL OR p.category_id = sqlc.narg('category_id'))
    AND (sqlc.narg('status_id')::uuid IS NULL OR p.status_id = sqlc.narg('status_id'))
    AND (
        sqlc.narg('keyword')::text IS NULL
        OR p.name ILIKE '%' || sqlc.narg('keyword') || '%'
        OR p.description ILIKE '%' || sqlc.narg('keyword') || '%'
    )
    AND (
        NOT sqlc.arg('has_after')::boolean
        OR p.published_at < sqlc.narg('after_published_at')
        OR (p.published_at = sqlc.narg('after_published_at') AND p.id < sqlc.narg('after_id'))
    )
ORDER BY p.published_at DESC, p.id DESC
LIMIT sqlc.arg('limit_param');

-- name: ListPublishedProductsAsc :many
-- 公開済み商品を (published_at ASC, id ASC) の安定順で keyset ページネーション取得します。
-- category_id / status_id / keyword は指定時のみ絞り込み、has_after=true の場合は keyset 境界(after_*)より未来へ絞り込みます。
SELECT sqlc.embed(p)
FROM products AS p
WHERE p.published_at IS NOT NULL
    AND (sqlc.narg('category_id')::uuid IS NULL OR p.category_id = sqlc.narg('category_id'))
    AND (sqlc.narg('status_id')::uuid IS NULL OR p.status_id = sqlc.narg('status_id'))
    AND (
        sqlc.narg('keyword')::text IS NULL
        OR p.name ILIKE '%' || sqlc.narg('keyword') || '%'
        OR p.description ILIKE '%' || sqlc.narg('keyword') || '%'
    )
    AND (
        NOT sqlc.arg('has_after')::boolean
        OR p.published_at > sqlc.narg('after_published_at')
        OR (p.published_at = sqlc.narg('after_published_at') AND p.id > sqlc.narg('after_id'))
    )
ORDER BY p.published_at ASC, p.id ASC
LIMIT sqlc.arg('limit_param');
