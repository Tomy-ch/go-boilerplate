-- name: CreateProduct :exec
INSERT INTO products (
    id,
    name,
    description,
    price,
    quantity,
    stock_warning_threshold,
    status_id,
    category_id,
    published_at,
    image_path
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('description'),
    sqlc.arg('price'),
    sqlc.arg('quantity'),
    sqlc.arg('stock_warning_threshold'),
    sqlc.arg('status_id'),
    sqlc.arg('category_id'),
    sqlc.arg('published_at'),
    sqlc.arg('image_path')
);
