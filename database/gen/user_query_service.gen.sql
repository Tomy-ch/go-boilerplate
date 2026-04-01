
-- === source: database/dml/query_service/user/select_users_by_keyword.sql ===
-- name: ListUsersByKeywords :many
SELECT sqlc.embed(u)
FROM users AS u
WHERE CASE sqlc.arg('active_state')
        WHEN 'active' THEN u.deleted_at IS NULL
        WHEN 'deleted' THEN u.deleted_at IS NOT NULL
        ELSE TRUE
    END
    AND u.search_text ILIKE ALL(sqlc.arg('patterns_param')::TEXT [])
ORDER BY u.created_at DESC
LIMIT sqlc.arg('limit_param') OFFSET sqlc.arg('offset_param');
