-- name: CountProducts :one
-- 登録済みの商品総数と、そのうち公開済みの商品数を返します。
-- 「公開中」を定義するのは Product.IsPublished で、FILTER 句はその実行形です。片方だけ変更しないこと。
SELECT
    COUNT(*)::BIGINT AS total_count,
    (COUNT(*) FILTER (WHERE published_at IS NOT NULL))::BIGINT AS published_count
FROM products;
