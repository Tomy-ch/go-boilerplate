-- name: UpdateProduct :one
-- 楽観ロック条件付きで商品を更新し、採番後のバージョンを返します。
-- lock_version の加算は DB が行い、採番の権威を単一箇所に置きます。
-- WHERE の lock_version 一致が並行更新による lost update を防ぐ本体で、読み込み後に他トランザクションが
-- 更新していた場合は該当行なし（0 行）で返り、呼び出し側が衝突として扱います。
UPDATE products
SET
    name = sqlc.arg('name'),
    description = sqlc.arg('description'),
    price = sqlc.arg('price'),
    quantity = sqlc.arg('quantity'),
    stock_warning_threshold = sqlc.arg('stock_warning_threshold'),
    status_id = sqlc.arg('status_id'),
    category_id = sqlc.arg('category_id'),
    published_at = sqlc.arg('published_at'),
    lock_version = products.lock_version + 1,
    updated_at = NOW()
WHERE products.id = sqlc.arg('id')
    AND products.lock_version = sqlc.arg('current_version')
RETURNING products.lock_version;
