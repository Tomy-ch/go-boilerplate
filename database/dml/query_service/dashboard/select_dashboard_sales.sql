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
