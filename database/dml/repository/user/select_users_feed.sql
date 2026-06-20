-- name: ListUsersFeedFirst :many
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.deleted_at IS NULL
ORDER BY u.created_at DESC, u.id DESC
LIMIT sqlc.arg('limit_param');

-- name: ListUsersFeedAfter :many
-- keyset 比較は (created_at, id) のタプル比較と等価だが、sqlc が各プレースホルダの型を
-- 比較対象カラムから正しく推論できるよう（特に id を uuid として推論させるため）、
-- 行値コンストラクタではなく展開形で記述する。
SELECT sqlc.embed(u)
FROM users AS u
WHERE u.deleted_at IS NULL
    AND (
        u.created_at < sqlc.arg('after_created_at')
        OR (u.created_at = sqlc.arg('after_created_at') AND u.id < sqlc.arg('after_id'))
    )
ORDER BY u.created_at DESC, u.id DESC
LIMIT sqlc.arg('limit_param');
