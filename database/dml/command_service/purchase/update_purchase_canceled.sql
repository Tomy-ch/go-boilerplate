-- name: UpdatePurchaseCanceled :exec
-- 購入をキャンセル状態へ更新する。status_id は code から解決し（seed UUID を焼き込まない）、
-- canceled_at / updated_at を NOW() でセットする（status と timestamp の同時セット・ADR-0028）。
-- 対象行は呼び出し側が FOR UPDATE で取得・検証済みのため、遷移可否ガードは付けない（ドメインが SoT）。
UPDATE purchases
SET
    status_id = (
        SELECT ps.id FROM purchase_statuses AS ps
        WHERE ps.code = @status_code
    ),
    canceled_at = NOW(),
    updated_at = NOW()
WHERE purchases.id = @id;
