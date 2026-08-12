-- name: LockUserShareByID :one
-- ID からユーザーを 1 件、悲観ロック（FOR SHARE）して取得する。不存在は 0 行（NotFound）。
-- 退会済みを除外しないこと — ロックは機構で、在籍かどうかの判定はドメイン（User.IsActive）が持つ。
-- 退会との直列化は docs/spec/purchase/usecase.md の CreatePurchase を参照。
-- ADR-0033 (ordered-pessimistic-row-locks)。
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.id = sqlc.arg('user_id_param')
FOR SHARE;
