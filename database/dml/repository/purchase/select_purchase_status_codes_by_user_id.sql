-- name: SelectPurchaseStatusCodesByUserID :many
-- 指定ユーザーの購入が取っているステータス code を重複なく返す。
-- 進行中かどうかの判定はドメイン（Status.IsTerminal の否定）が行うため、ここでは業務条件で絞り込まない。
-- 重複を除くため行数はステータスの種類数で頭打ちになり、購入件数には比例しない。
-- ステータスは購入ステータスマスタとの結合で解決する（JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
SELECT DISTINCT ps.code
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.user_id = sqlc.arg('user_id');
