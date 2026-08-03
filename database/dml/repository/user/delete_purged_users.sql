-- name: DeleteUserIdentitiesByUserIDs :exec
-- 物理削除するユーザーに従属する認証アイデンティティを削除する。users より先に消して FK 違反を避ける。
-- 対象は論理削除済みのユーザーに限る。users 側のガードと条件を揃えないと、
-- 削除されないユーザーの従属行だけが失われ、ログインできない生存アカウントが残る。
DELETE FROM user_identities
WHERE user_id IN (
        SELECT u.id
        FROM users AS u
        WHERE u.id = ANY(sqlc.arg('user_ids')::UUID [])
            AND u.deleted_at IS NOT NULL
    );

-- name: DeleteUserRolesByUserIDs :exec
-- 物理削除するユーザーに従属するロール割り当てを削除する。users より先に消して FK 違反を避ける。
-- 対象を論理削除済みのユーザーに限る理由は DeleteUserIdentitiesByUserIDs と同じ。
DELETE FROM user_roles
WHERE user_id IN (
        SELECT u.id
        FROM users AS u
        WHERE u.id = ANY(sqlc.arg('user_ids')::UUID [])
            AND u.deleted_at IS NOT NULL
    );

-- name: DeleteUsersByIDs :execrows
-- 物理削除の対象ユーザーを削除し、削除件数を返す。従属行の削除後に呼ばれる前提で、参照の残存はここでは検査しない。
-- 論理削除済みであることは、呼び手が候補列挙を誤っても現役ユーザーを不可逆に消さないための最終防壁として、
-- 保持期間の判定（Usecase の責務）とは別にここでも検査する。
DELETE FROM users
WHERE id = ANY(sqlc.arg('ids')::UUID [])
    AND deleted_at IS NOT NULL;
