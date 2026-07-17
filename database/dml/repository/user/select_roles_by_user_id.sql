-- name: GetUserRolesByUserID :many
SELECT
    r.id,
    r.name,
    r.code
FROM user_roles AS ur
INNER JOIN roles AS r ON ur.role_id = r.id
WHERE ur.user_id = sqlc.arg('user_id_param')
ORDER BY r.code;
