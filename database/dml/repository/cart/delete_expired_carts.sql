-- name: DeleteExpiredCarts :execrows
-- 削除の対象を決める述語の定義はドメインの Cart.IsExpired が持ち、この WHERE はその実行形である。
-- 有効期限ちょうどの時点は期限切れではない（< であって <= ではない）。片方だけを変更してはならない。
DELETE FROM carts
WHERE carts.id IN (
        SELECT c.id
        FROM carts AS c
        WHERE c.expires_at < sqlc.arg('now')
        ORDER BY c.expires_at, c.id
        LIMIT sqlc.arg('row_limit')
    );
