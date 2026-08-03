-- name: DeleteUserIdentitiesByUserIDs :exec
-- 物理削除するユーザーに従属する認証アイデンティティを削除する。users より先に消して FK 違反を避ける。
DELETE FROM user_identities
WHERE user_id = ANY(sqlc.arg('user_ids')::UUID []);

-- name: DeleteUserRolesByUserIDs :exec
-- 物理削除するユーザーに従属するロール割り当てを削除する。users より先に消して FK 違反を避ける。
DELETE FROM user_roles
WHERE user_id = ANY(sqlc.arg('user_ids')::UUID []);

-- name: DeleteUsersByIDs :execrows
-- 物理削除の対象ユーザーを削除し、削除件数を返す。従属行の削除後に呼ばれる前提で、参照の残存はここでは検査しない。
DELETE FROM users
WHERE id = ANY(sqlc.arg('ids')::UUID []);
