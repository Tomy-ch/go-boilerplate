-- name: LockUserShareByID :one
-- ID からユーザーを 1 件、悲観ロック（FOR SHARE）して取得する。
-- ロックは機構であり、取得した状態が在籍かどうかの判定はドメイン（User.IsActive）が行うため、
-- ここでは退会済みを除外しない。
-- 共有ロック同士は両立するため同一ユーザーの並行取得は直列化されず、退会が取る FOR UPDATE とだけ
-- 衝突する。これにより「退会の判定通過 → 購入の成立 → 退会の確定」の順序が成立しなくなる。
-- 不存在は 0 行（NotFound）。
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.id = sqlc.arg('user_id_param')
FOR SHARE;
