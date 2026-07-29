-- name: GetProductCategoryByID :one
SELECT
    pc.id,
    pc.name,
    pc.code,
    pc.sort_key
FROM product_categories AS pc
WHERE pc.id = sqlc.arg('id');
