-- name: UpdatePurchaseShipped :exec
-- 購入を発送済み状態へ更新する。配送追跡は扱わないため単一集約（purchases）のみを更新し、在庫操作は伴わない。
-- status_id は code から解決し（seed UUID を焼き込まない）、shipped_at はドメインが決定した時刻（引数）を書き込み、
-- イベント payload・レスポンスと同一時刻に揃える。対象行は呼び出し側が FOR UPDATE で取得・検証済みのため、
-- 遷移可否ガードは付けない（ドメインが SoT）。
UPDATE purchases
SET
    status_id = (
        SELECT ps.id FROM purchase_statuses AS ps
        WHERE ps.code = @status_code
    ),
    shipped_at = @shipped_at,
    updated_at = NOW()
WHERE purchases.id = @id;
