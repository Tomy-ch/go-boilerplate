-- name: DeleteUserIdentitiesByUserIDs :exec
-- users より先に呼ぶこと（FK 違反を避ける）。論理削除済みに限る条件は DeleteUsersByIDs の
-- WHERE と揃えること — ずれると、削除されないユーザーの従属行だけが失われる。
DELETE FROM user_identities
WHERE user_id IN (
        SELECT u.id
        FROM users AS u
        WHERE u.id = ANY(sqlc.arg('user_ids')::UUID[])
            AND u.deleted_at IS NOT NULL
    );

-- name: DeleteUserRolesByUserIDs :exec
-- users より先に呼ぶこと（FK 違反を避ける）。論理削除済みに限る理由は
-- DeleteUserIdentitiesByUserIDs と同じ。
DELETE FROM user_roles
WHERE user_id IN (
        SELECT u.id
        FROM users AS u
        WHERE u.id = ANY(sqlc.arg('user_ids')::UUID[])
            AND u.deleted_at IS NOT NULL
    );

-- name: DeleteUsersByIDs :execrows
-- 削除件数を返す。従属行の削除後に呼ぶこと（参照の残存はここでは検査しない）。
-- 論理削除済みを永続化側でも検査する理由は docs/spec/domain/user.md の PurgeByIDs を参照。
DELETE FROM users
WHERE id = ANY(sqlc.arg('ids')::UUID[])
    AND deleted_at IS NOT NULL;

-- name: DeleteCouponsByUserIDs :exec
-- users より先に呼ぶこと（FK 違反を避ける）。論理削除済みに限る理由は
-- DeleteUserIdentitiesByUserIDs と同じ。
-- 物理削除の対象は購入を 1 件も持たない利用者に限られる。控えが存在しないため、
-- 「値引きの理由を控えとの結合で解決する」という保持の根拠がこの対象には届かない
-- （docs/spec/domain/coupon.md の Notes）。
DELETE FROM coupons
WHERE user_id IN (
        SELECT u.id
        FROM users AS u
        WHERE u.id = ANY(sqlc.arg('user_ids')::UUID[])
            AND u.deleted_at IS NOT NULL
    );
