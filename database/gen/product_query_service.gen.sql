
-- === source: database/dml/query_service/product/select_existing_image_paths.sql ===
-- name: ListExistingProductImagePaths :many
-- 与えた画像パスのうち、いずれかの商品が実際に参照しているものを返す。
-- 未参照オブジェクトの回収（product-image-gc）で「消してよいか」を判定する取得元で、
-- ここに現れなかったパスが孤児にあたる。
-- 論理削除された画像は差し替え履歴であって現在の参照ではないため、生存行だけを参照元として数える。
SELECT DISTINCT pi.image_path
FROM product_images AS pi
WHERE pi.image_path = ANY(sqlc.arg('image_paths')::TEXT[])
    AND pi.deleted_at IS NULL;

-- === source: database/dml/query_service/product/select_product_ranking.sql ===
-- name: ListProductQuantityRanking :many
-- 購入明細を商品単位で集計し、販売数量の降順で上位 limit_count 件を返します。
-- 公開済み（published_at 非 NULL）の商品のみを対象とし、非公開・未存在は集計から除外します。
-- product.IsPublished と同値（database/dml/query_service/README.md 参照）。
-- published_at を返すのは呼び出し側が返却行を同じ定義で突き合わせるためで、表示には用いません。
-- キャンセル済み（canceled_at 設定済み）の購入は除外し、未払いの購入は含みます。
-- Purchase.IsCanceled と同値（database/dml/query_service/README.md 参照）。
-- 注文日時は半開区間 [ordered_after, ordered_before)（internal/usecase/tools/timewindow/README.md）で絞り込みます。
-- 同一販売数量は商品 ID の昇順で安定的に並べます。
SELECT
    p.id AS product_id,
    p.name,
    p.price,
    p.published_at,
    SUM(pd.quantity)::BIGINT AS sold_quantity
FROM purchase_details AS pd
INNER JOIN purchases AS pur ON pd.purchase_id = pur.id AND pur.canceled_at IS NULL
INNER JOIN products AS p ON pd.product_id = p.id AND p.published_at IS NOT NULL
WHERE
    (
        pur.ordered_at >= sqlc.narg('ordered_after')
        OR sqlc.narg('ordered_after') IS NULL
    )
    AND (
        pur.ordered_at < sqlc.narg('ordered_before')
        OR sqlc.narg('ordered_before') IS NULL
    )
GROUP BY p.id
ORDER BY sold_quantity DESC, p.id ASC
LIMIT sqlc.arg('limit_count');

-- name: ListProductAmountRanking :many
-- 購入明細を商品単位で集計し、売上金額（単価 × 数量の総和）の降順で上位 limit_count 件を返します。
-- 母集団は ListProductQuantityRanking と逐語的に同一で、集計する指標と並び順だけが異なります
-- （同じ期間を指定した 2 つの口が同じ母集団を集計することの担保）。
-- 金額は価格スケールの正確な decimal で、決済スケールへは丸めません（ADR-0038）。
-- 同一売上金額は商品 ID の昇順で安定的に並べます。
SELECT
    p.id AS product_id,
    p.name,
    p.price,
    p.published_at,
    SUM(pd.unit_price * pd.quantity)::NUMERIC AS sales_amount
FROM purchase_details AS pd
INNER JOIN purchases AS pur ON pd.purchase_id = pur.id AND pur.canceled_at IS NULL
INNER JOIN products AS p ON pd.product_id = p.id AND p.published_at IS NOT NULL
WHERE
    (
        pur.ordered_at >= sqlc.narg('ordered_after')
        OR sqlc.narg('ordered_after') IS NULL
    )
    AND (
        pur.ordered_at < sqlc.narg('ordered_before')
        OR sqlc.narg('ordered_before') IS NULL
    )
GROUP BY p.id
ORDER BY sales_amount DESC, p.id ASC
LIMIT sqlc.arg('limit_count');
