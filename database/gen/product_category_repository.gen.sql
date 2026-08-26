
-- === source: database/dml/repository/product_category/select_all_product_categories.sql ===
-- name: GetProductCategoryDomainAll :many
SELECT
    pc.id,
    pc.name,
    pc.code,
    pc.sort_key
FROM product_categories AS pc
ORDER BY pc.sort_key ASC;

-- === source: database/dml/repository/product_category/select_product_category_by_id.sql ===
-- name: GetProductCategoryByID :one
SELECT
    pc.id,
    pc.name,
    pc.code,
    pc.sort_key
FROM product_categories AS pc
WHERE pc.id = sqlc.arg('id');
