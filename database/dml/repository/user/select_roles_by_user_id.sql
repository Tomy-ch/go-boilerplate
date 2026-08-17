-- name: GetUserRolesByUserID :many
-- 指定ユーザーのロールをマスタの表示順（sort_key 昇順）で返す。並び順の出所は code ではない。
SELECT
    r.id,
    r.name,
    r.code
FROM user_roles AS ur
INNER JOIN roles AS r ON ur.role_id = r.id
WHERE ur.user_id = sqlc.arg('user_id_param')
ORDER BY r.sort_key;
