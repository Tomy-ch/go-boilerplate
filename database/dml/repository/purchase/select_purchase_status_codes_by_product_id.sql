-- name: SelectPurchaseStatusCodesByProductID :many
-- 指定商品を明細に持つ購入が取っているステータスの code を重複なく返す。
-- 進行中かどうかで絞らないのは、その判定を購入集約（Status.IsTerminal）が持つためで、
-- SQL 側に同じ規則を書き写さない（理由は docs/spec/domain/purchase.md の FindStatusesByProductID）。
SELECT DISTINCT ps.code
FROM purchases AS p
INNER JOIN purchase_details AS pd ON p.id = pd.purchase_id
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE pd.product_id = sqlc.arg('product_id');
