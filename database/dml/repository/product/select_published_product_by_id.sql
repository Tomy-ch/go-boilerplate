-- name: GetPublishedProductByID :one
-- ID から公開中の単一商品を取得します。
-- 公開範囲の定義は一覧取得（ListPublishedProducts*）と同一述語（published_at 非 NULL）で、
-- 非公開・未存在はいずれも該当なし（sql.ErrNoRows）に落ち、infra 層で NotFound へ正規化されます。
SELECT sqlc.embed(p)
FROM products AS p
WHERE p.id = sqlc.arg('product_id_param')
    AND p.published_at IS NOT NULL;
