-- name: UpdatePurchaseDelivered :exec
-- 購入を配達済み状態へ更新する。status_id は code から解決する。delivered_at はドメインが決定した時刻（引数）を
-- 書き込み、イベント payload・レスポンスと同一時刻に揃える。
-- 遷移可否ガードは付けない（理由は docs/spec/purchase/domain.md の Repository Methods）。
UPDATE purchases
SET
    status_id = (
        SELECT ps.id FROM purchase_statuses AS ps
        WHERE ps.code = @status_code
    ),
    delivered_at = @delivered_at,
    updated_at = NOW()
WHERE purchases.id = @id;
