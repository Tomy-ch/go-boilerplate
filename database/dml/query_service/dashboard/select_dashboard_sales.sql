-- name: SummarizeDashboardSales :one
-- 指定期間に注文された購入の売上合計と件数を返します。
-- 期間は [ordered_after, ordered_before) の半開区間で、境界時刻の算出はインフラ層が担います。
-- キャンセル済み（canceled_at 設定済み）の購入は除外し、未払い（paid_at 未設定）の購入は含めます
-- （商品売上ランキングと同一の母集団）。
-- 「キャンセル済み」の定義はドメイン（Purchase.IsCanceled）が持ち、この条件はその実行形です。
-- 述語が見るのは status ですが、両者は再構築時の不変条件で等価に縛られています。
-- 除外は減算的な基準で、落とした行は結果に現れないため呼び出し側では照合できません。退行はテストが固定します。
-- 対象が 0 件のとき SUM は NULL を返すため、COALESCE でゼロ値へ畳み込みます。
SELECT
    COALESCE(SUM(total_amount), 0)::BIGINT AS sales_amount,
    COUNT(id)::BIGINT AS sales_count
FROM purchases
WHERE
    canceled_at IS NULL
    AND ordered_at >= sqlc.arg('ordered_after')
    AND ordered_at < sqlc.arg('ordered_before');
