
-- === source: database/dml/repository/product_category/select_all_product_categories.sql ===
-- name: GetProductCategoryDomainAll :many
SELECT
    pc.id,
    pc.name,
    pc.code,
    pc.sort_key
FROM product_categories AS pc
ORDER BY pc.sort_key ASC;
