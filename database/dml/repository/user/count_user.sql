-- name: CountUsersByActiveState :one
SELECT COUNT(*)
FROM users AS u
WHERE CASE sqlc.arg('active_state')
        WHEN 'active' THEN u.deleted_at IS NULL
        WHEN 'deleted' THEN u.deleted_at IS NOT NULL
        ELSE TRUE
    END;
