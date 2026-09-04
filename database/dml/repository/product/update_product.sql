-- name: UpdateProduct :one
-- 楽観ロック条件付きで商品を更新し、採番後のバージョンを返します。
-- 挙動の詳細は docs/spec/domain/product.md の Product.Update を参照。
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
    discontinued_at = sqlc.arg('discontinued_at'),
    lock_version = products.lock_version + 1,
    updated_at = NOW()
WHERE products.id = sqlc.arg('id')
    AND products.lock_version = sqlc.arg('current_version')
RETURNING products.lock_version;
