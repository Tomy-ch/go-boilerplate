-- name: CountDiscontinueImpactInProgressPurchases :one
-- 廃番を阻む進行中の購入の件数を返す。
-- 「進行中」は購入集約が定義する（Status.IsTerminal の否定）ため、終端のステータス code を
-- 呼び出し側から受け取り、SQL 側に規則を書き写さない。
-- 行はロックしないため、返した値は返した瞬間から古くなる。実行時に改めて判定される。
SELECT COUNT(DISTINCT p.id)
FROM purchases AS p
INNER JOIN purchase_details AS pd ON pd.purchase_id = p.id
INNER JOIN purchase_statuses AS ps ON ps.id = p.status_id
WHERE pd.product_id = sqlc.arg('product_id')
    AND NOT (ps.code = ANY(sqlc.arg('terminal_status_codes')::SMALLINT[]));
