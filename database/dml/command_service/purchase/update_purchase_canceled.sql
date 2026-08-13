-- name: UpdatePurchaseCanceled :exec
-- 購入をキャンセル状態へ更新する。status_id は code から解決する。canceled_at はドメインが決定した
-- 時刻（引数）を書き込み、イベント payload・レスポンスと同一時刻に揃える。
-- 遷移可否ガードは付けない（理由は internal/infrastructure/rdb/README.md の command_service 節）。
UPDATE purchases
SET
    status_id = (
        SELECT ps.id FROM purchase_statuses AS ps
        WHERE ps.code = @status_code
    ),
    canceled_at = @canceled_at,
    updated_at = NOW()
WHERE purchases.id = @id;
