-- name: ResolveUserByIdentity :one
SELECT
    u.id,
    u.deleted_at
FROM user_identities AS ui
INNER JOIN users AS u ON ui.user_id = u.id
WHERE ui.issuer = sqlc.arg('issuer_param')
    AND ui.subject = sqlc.arg('subject_param');
