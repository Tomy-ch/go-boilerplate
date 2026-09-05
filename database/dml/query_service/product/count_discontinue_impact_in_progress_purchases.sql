-- name: CountDiscontinueImpactInProgressPurchases :one
-- 廃番を阻む進行中の購入の件数を返す。
-- 終端のステータス code を呼び出し側から受け取る（進行中の定義は
-- docs/spec/domain/purchase.md の FindStatusesByProductID を参照）。
-- 見積もりの古さは docs/spec/usecase/product.md の GetDiscontinueImpact を参照。
SELECT COUNT(DISTINCT p.id)
FROM purchases AS p
INNER JOIN purchase_details AS pd ON p.id = pd.purchase_id
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE pd.product_id = sqlc.arg('product_id')
    AND NOT (ps.code = ANY(sqlc.arg('terminal_status_codes')::SMALLINT[]));
