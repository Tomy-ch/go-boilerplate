-- name: ListUserIDsWithPurchases :many
-- 与えたユーザー ID のうち、購入を 1 件以上持つものを返す。購入は独立集約のため、
-- ユーザー側の絞り込みと結合せず ID 群の照会として切り出す（docs/rules.md の Repository / QueryService Rules）。
SELECT DISTINCT user_id
FROM purchases
WHERE user_id = ANY(sqlc.arg('user_ids')::UUID []);
