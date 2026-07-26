-- name: SummarizePurchasesByUserID :many
-- 指定ユーザーの購入をステータス単位に集計し、購入ステータスマスタの表示順（sort_key 昇順）で返します。
-- 所有権は user_id の等値条件で閉じるため、他ユーザーの購入は集計に混入しません。
-- 既存の複合インデックス purchases (user_id, ordered_at DESC, id DESC) の先頭列で絞り込みます。
-- キャンセル済み（canceled_at 設定済み）の購入も対象に含めます。キャンセルはステータス別内訳の
-- 1 要素として返るため、除外すると内訳と総計が食い違います。
-- 総件数・合計金額はステータス別の集計値をユースケース層で畳み込みます（単一スナップショットで整合させるため）。
-- ステータス名は購入ステータスマスタとの結合で解決します（購入集約に属する固定参照マスタへの一意な等結合）。
SELECT
    ps.id AS status_id,
    ps.name AS status_name,
    COUNT(p.id)::BIGINT AS purchase_count,
    COALESCE(SUM(p.total_amount), 0)::BIGINT AS total_amount
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.user_id = sqlc.arg('user_id')
GROUP BY ps.id, ps.name, ps.sort_key
ORDER BY ps.sort_key ASC;
