
-- === source: database/dml/repository/product_status/select_all_product_statuses.sql ===
-- name: GetProductStatusDomainAll :many
SELECT
    ps.id,
    ps.name,
    ps.code,
    ps.sort_key
FROM product_statuses AS ps
ORDER BY ps.sort_key ASC;
