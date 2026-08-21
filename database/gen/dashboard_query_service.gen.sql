
-- === source: database/dml/query_service/dashboard/select_dashboard_purchase_status_counts.sql ===
-- name: CountDashboardPurchasesByStatus :many
-- 指定期間に注文された購入をステータス単位に集計し、購入ステータスマスタの表示順（sort_key 昇順）で返します。
-- 期間は半開区間 [ordered_after, ordered_before)（internal/usecase/tools/timewindow/README.md）です。
-- 売上集計と異なりキャンセル済みの購入も 1 ステータスとして含めます。
SELECT
    ps.id AS status_id,
    ps.code AS status_code,
    ps.name AS status_name,
    COUNT(p.id)::BIGINT AS purchase_count
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE
    (p.ordered_at >= sqlc.narg('ordered_after') OR sqlc.narg('ordered_after') IS NULL)
    AND (
        p.ordered_at < sqlc.narg('ordered_before')
        OR sqlc.narg('ordered_before') IS NULL
    )
GROUP BY ps.id, ps.code, ps.name, ps.sort_key
ORDER BY ps.sort_key ASC;

-- === source: database/dml/query_service/dashboard/select_dashboard_sales.sql ===
-- name: SummarizeDashboardSales :one
-- 指定期間に注文された購入の売上合計と件数を返します。
-- 期間は半開区間 [ordered_after, ordered_before)（internal/usecase/tools/timewindow/README.md）です。
-- キャンセル済み（canceled_at 設定済み）の購入は除外し、未払い（paid_at 未設定）の購入は含めます
-- （購入レベルの絞りは商品売上ランキングと同じだが、ランキングはさらに公開済み商品に限る）。
-- Purchase.IsCanceled と同値（database/dml/query_service/README.md 参照）。
-- 対象が 0 件のとき SUM は NULL を返すため、COALESCE でゼロ値へ畳み込みます。
SELECT
    COALESCE(SUM(total_amount), 0)::BIGINT AS sales_amount,
    COUNT(id)::BIGINT AS sales_count
FROM purchases
WHERE
    canceled_at IS NULL
    AND (
        ordered_at >= sqlc.narg('ordered_after') OR sqlc.narg('ordered_after') IS NULL
    )
    AND (
        ordered_at < sqlc.narg('ordered_before')
        OR sqlc.narg('ordered_before') IS NULL
    );
