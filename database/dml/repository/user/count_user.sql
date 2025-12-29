-- name: CountUsersByDeletedState :one
SELECT COUNT(*)
FROM users AS u
WHERE CASE sqlc.arg('deleted_state')::DELETED_STATE
        WHEN 'active' THEN u.deleted_at IS NULL
        WHEN 'deleted' THEN u.deleted_at IS NOT NULL
        ELSE TRUE
    END;
