
-- === source: database/dml/repository/purchase_status/select_all_purchase_statuses.sql ===
-- name: GetPurchaseStatusDomainAll :many
SELECT
    ps.id,
    ps.name,
    ps.code,
    ps.sort_key
FROM purchase_statuses AS ps
ORDER BY ps.sort_key ASC;
