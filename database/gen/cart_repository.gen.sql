
-- === source: database/dml/repository/cart/delete_cart.sql ===
-- name: DeleteCart :exec
-- カートを削除する。明細は外部キーの連鎖削除で除かれる。存在しない場合もエラーとしない。
DELETE FROM carts
WHERE carts.id = sqlc.arg('id');

-- === source: database/dml/repository/cart/delete_expired_carts.sql ===
-- name: DeleteExpiredCarts :execrows
-- 有効期限を過ぎたカートを最大 limit 件削除し、削除件数を返す。1 回で消し切ることを意図せず、
-- 件数上限で区切って繰り返し呼ばれる。
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

-- === source: database/dml/repository/cart/insert_cart.sql ===
-- name: CreateCart :exec
-- カートを新規登録する。user_id / session_token の一意制約違反は呼び出し側が衝突として扱う。
INSERT INTO carts (
    id,
    user_id,
    session_token,
    expires_at
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('user_id'),
    sqlc.arg('session_token'),
    sqlc.arg('expires_at')
);

-- name: CreateCartItem :exec
-- カート明細を 1 件登録する。同一カート内の同一商品は一意（cart_items_cart_id_product_id_unique）。
INSERT INTO cart_items (
    id,
    cart_id,
    product_id,
    quantity,
    last_seen_price,
    added_at
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('cart_id'),
    sqlc.arg('product_id'),
    sqlc.arg('quantity'),
    sqlc.arg('last_seen_price'),
    sqlc.arg('added_at')
);

-- === source: database/dml/repository/cart/lock_cart_by_id.sql ===
-- name: LockCartByID :one
-- ID からカートを 1 件、悲観ロック（FOR UPDATE）して取得する。同一カートへの並行更新を、取得から
-- commit まで直列化する。複数件をロックする場合の順序は呼び出し側が ID 昇順で固定する。
-- 存在しない場合は 0 行（NotFound）。
SELECT sqlc.embed(c)
FROM carts AS c
WHERE c.id = sqlc.arg('id')
FOR UPDATE;

-- === source: database/dml/repository/cart/select_cart_by_owner_id.sql ===
-- name: GetCartByOwnerID :one
-- 所有者からカートを 1 件取得する。ユーザー 1 人につきカートは高々 1 件（carts_user_id_unique）。
-- 存在しない場合は 0 行（NotFound）。
-- 有効期限で絞らないのは意図的で、期限切れかどうかの判定はドメインの述語が持つ。ここで取り除くと
-- 取り除かれた行は結果に現れず、不在を観測できないため、突き合わせ検証の余地が消える。
SELECT sqlc.embed(c)
FROM carts AS c
WHERE c.user_id = sqlc.arg('user_id');

-- === source: database/dml/repository/cart/select_cart_by_session_token.sql ===
-- name: GetCartBySessionToken :one
-- セッショントークンからカートを 1 件取得する。存在しない場合は 0 行（NotFound）。
-- 所有者が確定したカートは session_token を持たない（carts_owner_exclusive）ため、この経路では引けない。
-- 有効期限で絞らない理由は GetCartByOwnerID と同じ。
SELECT sqlc.embed(c)
FROM carts AS c
WHERE c.session_token = sqlc.arg('session_token');

-- === source: database/dml/repository/cart/select_cart_items_by_cart_id.sql ===
-- name: ListCartItemsByCartID :many
-- カート ID から明細を取得する。並びは追加日時の昇順、同時刻は ID 昇順で決着させる。
-- 明示するのは表示の決定性とテストの再現性のためで、切り捨ての順序をこの並びに依存させてはならない
-- （その判定は追加日時の値から集約が行う）。
SELECT sqlc.embed(ci)
FROM cart_items AS ci
WHERE ci.cart_id = sqlc.arg('cart_id')
ORDER BY ci.added_at, ci.id;

-- === source: database/dml/repository/cart/update_cart.sql ===
-- name: UpdateCart :execrows
-- カートの親行を更新し、更新件数を返す。0 件は対象が存在しないことを意味し、呼び出し側が NotFound
-- として扱う。所有者の確定（user_id のセットと session_token の NULL 化）もこの 1 本で反映される。
UPDATE carts
SET
    user_id = sqlc.arg('user_id'),
    session_token = sqlc.arg('session_token'),
    expires_at = sqlc.arg('expires_at'),
    updated_at = NOW()
WHERE carts.id = sqlc.arg('id');

-- name: UpsertCartItem :exec
-- 明細を商品ごとに登録または置換する。added_at は SET の対象に含めない。置換のたびに更新すると
-- 「いつカートへ入ったか」が失われ、切り捨ての順序を決める基準が壊れるため。
INSERT INTO cart_items (
    id,
    cart_id,
    product_id,
    quantity,
    last_seen_price,
    added_at
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('cart_id'),
    sqlc.arg('product_id'),
    sqlc.arg('quantity'),
    sqlc.arg('last_seen_price'),
    sqlc.arg('added_at')
)
ON CONFLICT ON CONSTRAINT cart_items_cart_id_product_id_unique DO UPDATE
    SET
        quantity = excluded.quantity,
        last_seen_price = excluded.last_seen_price,
        updated_at = NOW();

-- name: DeleteCartItemsNotIn :exec
-- 指定した商品の集合に含まれない明細を取り除く。集約が保持しなくなった明細を落とすためのもので、
-- 空の集合を渡した場合はそのカートの明細をすべて削除する。
DELETE FROM cart_items
WHERE cart_items.cart_id = sqlc.arg('cart_id')
    AND NOT (cart_items.product_id = ANY(sqlc.arg('product_ids')::UUID []));
