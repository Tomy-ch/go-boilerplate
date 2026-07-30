
-- === source: database/dml/query_service/dashboard/select_dashboard_purchase_status_counts.sql ===
-- name: CountDashboardPurchasesByStatus :many
-- 指定期間に注文された購入をステータス単位に集計し、購入ステータスマスタの表示順（sort_key 昇順）で返します。
-- 期間は [ordered_after, ordered_before) の半開区間で、境界時刻の算出はインフラ層が担います。
-- 売上集計と異なりキャンセル済みの購入も 1 ステータスとして含めます。
-- ステータス名は購入ステータスマスタとの結合で解決します（購入集約に属する固定参照マスタへの一意な等結合）。
SELECT
    ps.id AS status_id,
    ps.name AS status_name,
    COUNT(p.id)::BIGINT AS purchase_count
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE
    p.ordered_at >= sqlc.arg('ordered_after')
    AND p.ordered_at < sqlc.arg('ordered_before')
GROUP BY ps.id, ps.name, ps.sort_key
ORDER BY ps.sort_key ASC;

-- === source: database/dml/query_service/dashboard/select_dashboard_sales.sql ===
-- name: SummarizeDashboardSales :one
-- 指定期間に注文された購入の売上合計と件数を返します。
-- 期間は [ordered_after, ordered_before) の半開区間で、境界時刻の算出はインフラ層が担います。
-- キャンセル済み（canceled_at 設定済み）の購入は除外し、未払い（paid_at 未設定）の購入は含めます
-- （商品売上ランキングと同一の母集団）。
-- 対象が 0 件のとき SUM は NULL を返すため、COALESCE でゼロ値へ畳み込みます。
SELECT
    COALESCE(SUM(total_amount), 0)::BIGINT AS sales_amount,
    COUNT(id)::BIGINT AS sales_count
FROM purchases
WHERE
    canceled_at IS NULL
    AND ordered_at >= sqlc.arg('ordered_after')
    AND ordered_at < sqlc.arg('ordered_before');
