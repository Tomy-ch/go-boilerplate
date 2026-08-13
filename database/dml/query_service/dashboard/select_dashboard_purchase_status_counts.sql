-- name: CountDashboardPurchasesByStatus :many
-- 指定期間に注文された購入をステータス単位に集計し、購入ステータスマスタの表示順（sort_key 昇順）で返します。
-- 期間は [ordered_after, ordered_before) の半開区間です。
-- 売上集計と異なりキャンセル済みの購入も 1 ステータスとして含めます。
-- ステータス名は購入ステータスマスタとの結合で解決します。
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
