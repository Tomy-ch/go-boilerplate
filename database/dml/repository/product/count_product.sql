-- name: CountProducts :one
-- 登録済みの商品総数と、そのうち公開済み（published_at 設定済み）の商品数を返します。
SELECT
    COUNT(*)::BIGINT AS total_count,
    (COUNT(*) FILTER (WHERE published_at IS NOT NULL))::BIGINT AS published_count
FROM products;
