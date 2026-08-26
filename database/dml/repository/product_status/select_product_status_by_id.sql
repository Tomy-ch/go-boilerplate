-- name: GetProductStatusByID :one
SELECT
    ps.id,
    ps.name,
    ps.code,
    ps.sort_key
FROM product_statuses AS ps
WHERE ps.id = sqlc.arg('id');
