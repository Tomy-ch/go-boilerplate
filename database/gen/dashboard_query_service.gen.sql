
-- === source: database/dml/query_service/dashboard/select_dashboard_purchase_status_counts.sql ===
-- name: CountDashboardPurchasesByStatus :many
-- 指定期間に注文された購入をステータス単位に集計し、購入ステータスマスタの表示順（sort_key 昇順）で返します。
-- 期間は [ordered_after, ordered_before) の半開区間です。
-- 売上集計と異なりキャンセル済みの購入も 1 ステータスとして含めます。
SELECT
    ps.id AS status_id,
    ps.code AS status_code,
    ps.name AS status_name,
    COUNT(p.id)::BIGINT AS purchase_count
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE
    p.ordered_at >= sqlc.arg('ordered_after')
    AND p.ordered_at < sqlc.arg('ordered_before')
GROUP BY ps.id, ps.code, ps.name, ps.sort_key
ORDER BY ps.sort_key ASC;

-- === source: database/dml/query_service/dashboard/select_dashboard_sales.sql ===
-- name: SummarizeDashboardSales :one
-- 指定期間に注文された購入の売上合計と件数を返します。
-- 期間は [ordered_after, ordered_before) の半開区間です。
-- キャンセル済み（canceled_at 設定済み）の購入は除外し、未払い（paid_at 未設定）の購入は含めます
-- （商品売上ランキングと同一の母集団）。
-- 「キャンセル済み」の定義はドメイン（Purchase.IsCanceled）が持ち、この条件はその実行形です。
-- 述語が見るのは status ですが、両者は再構築時の不変条件で等価に縛られています。
-- 対象が 0 件のとき SUM は NULL を返すため、COALESCE でゼロ値へ畳み込みます。
SELECT
    COALESCE(SUM(total_amount), 0)::BIGINT AS sales_amount,
    COUNT(id)::BIGINT AS sales_count
FROM purchases
WHERE
    canceled_at IS NULL
    AND ordered_at >= sqlc.arg('ordered_after')
    AND ordered_at < sqlc.arg('ordered_before');
