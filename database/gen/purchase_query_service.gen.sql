
-- === source: database/dml/query_service/purchase/select_purchase_detail_by_code.sql ===
-- name: GetPurchaseDetailForUser :one
-- 認証主体の購入本体 1 件を購入コードで取得する。
-- 所有権は WHERE 述語（user_id 一致）で担保し、他人・不存在はいずれも 0 行（NotFound で秘匿）。
-- 支払い日時（paid_at）は未支払いなら NULL、キャンセル日時（canceled_at）は未キャンセルなら NULL。
SELECT
    p.id,
    p.code,
    p.user_id,
    ps.id AS status_id,
    ps.code AS status_code,
    ps.name AS status_name,
    p.subtotal_amount,
    p.tax_amount,
    p.shipping_fee,
    p.total_amount,
    p.ordered_at,
    p.paid_at,
    p.canceled_at
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.code = @code AND p.user_id = @user_id;

-- name: ListPurchaseDetailItemsForUser :many
-- 購入明細を products との結合で商品名込みに取得する（集約跨ぎの read 投影）。
-- 本体行から得た購入 ID で引くため、所有権は本体クエリ側で既に閉じている。
-- product_id は FK 制約により products と常に結合可能。id 昇順で安定整列する。
SELECT
    d.product_id,
    pr.name AS product_name,
    d.quantity,
    d.unit_price
FROM purchase_details AS d
INNER JOIN products AS pr ON d.product_id = pr.id
WHERE d.purchase_id = @purchase_id_param
ORDER BY d.id;

-- === source: database/dml/query_service/purchase/select_purchase_summary.sql ===
-- name: SummarizePurchasesByUserID :many
-- 指定ユーザーの購入をステータス単位に集計し、購入ステータスマスタの表示順（sort_key 昇順）で返します。
-- 所有権は user_id の等値条件で閉じるため、他ユーザーの購入は集計に混入しません。
-- 既存の複合インデックス purchases (user_id, ordered_at DESC, id DESC) を使う。filter_by_period=false
-- のときは先頭列（user_id）のみが絞り込みに効きます。
-- キャンセル済み（canceled_at 設定済み）の購入は除外します。
-- 「キャンセル済み」の定義はドメイン（Purchase.IsCanceled）が持ち、この条件はその実行形です。
-- 述語が見るのは canceled_at ですが、両者は再構築時の不変条件で等価に縛られています。
-- filter_by_period=true の場合は注文日時が半開区間 [ordered_after, ordered_before) の購入だけを集計します。
-- 総件数・合計金額はこの結果行を畳み込んで算出します（単一スナップショットで整合させるため）。
SELECT
    ps.id AS status_id,
    ps.code AS status_code,
    ps.name AS status_name,
    COUNT(p.id)::BIGINT AS purchase_count,
    COALESCE(SUM(p.total_amount), 0)::BIGINT AS total_amount
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.user_id = sqlc.arg('user_id')
    AND p.canceled_at IS NULL
    AND (
        NOT sqlc.arg('filter_by_period')::BOOLEAN
        OR (
            p.ordered_at >= sqlc.narg('ordered_after')
            AND p.ordered_at < sqlc.narg('ordered_before')
        )
    )
GROUP BY ps.id, ps.code, ps.name, ps.sort_key
ORDER BY ps.sort_key ASC;

-- name: SumPurchaseItemsByUserID :one
-- 指定ユーザーの購入明細の金額合計（単価 × 数量の総和）を価格スケールの正確な decimal で返します。
-- 決済スケール（セント整数）へは丸めません（丸めは決済確定の 1 箇所のみ・ADR-0037）。
-- 母集団は SummarizePurchasesByUserID と同一（所有権・キャンセル除外・期間）です。
-- グループ化を要求されない既定の呼び出しはこのクエリだけで済み、商品単位の行を読みません。
-- 対象が 0 件のとき SUM は NULL を返すため、COALESCE でゼロ値へ畳み込みます。
SELECT COALESCE(SUM(pd.unit_price * pd.quantity), 0)::NUMERIC AS items_total
FROM purchase_details AS pd
INNER JOIN purchases AS p ON pd.purchase_id = p.id
WHERE p.user_id = sqlc.arg('user_id')
    AND p.canceled_at IS NULL
    AND (
        NOT sqlc.arg('filter_by_period')::BOOLEAN
        OR (
            p.ordered_at >= sqlc.narg('ordered_after')
            AND p.ordered_at < sqlc.narg('ordered_before')
        )
    );

-- name: SummarizePurchaseItemsByProductByUserID :many
-- 指定ユーザーの購入明細を商品単位に集計し、商品が属するカテゴリを添えて返します。
-- 商品は必ず 1 カテゴリに属する（products.category_id は NOT NULL）ため、行は商品単位で一意です。
-- カテゴリ単位だけを要求された場合も、呼び出し側はこの行をカテゴリで畳み込めば得られます。
-- 金額は価格スケールの正確な decimal で、決済スケールへは丸めません（ADR-0037）。
-- 母集団は SummarizePurchasesByUserID と同一（所有権・キャンセル除外・期間）です。
-- ランキングと異なり非公開商品も除外しません（購入時点の実績であり、現在の公開状態には依存しないため）。
-- 並びはカテゴリの表示順・商品名の昇順で安定させます（同名商品は商品 ID で分かれます）。
SELECT
    pc.name AS category_name,
    pr.id AS product_id,
    pr.name AS product_name,
    SUM(pd.unit_price * pd.quantity)::NUMERIC AS items_total
FROM purchase_details AS pd
INNER JOIN purchases AS p ON pd.purchase_id = p.id
INNER JOIN products AS pr ON pd.product_id = pr.id
INNER JOIN product_categories AS pc ON pr.category_id = pc.id
WHERE p.user_id = sqlc.arg('user_id')
    AND p.canceled_at IS NULL
    AND (
        NOT sqlc.arg('filter_by_period')::BOOLEAN
        OR (
            p.ordered_at >= sqlc.narg('ordered_after')
            AND p.ordered_at < sqlc.narg('ordered_before')
        )
    )
GROUP BY pc.id, pc.name, pc.sort_key, pr.id, pr.name
ORDER BY pc.sort_key ASC, pr.name ASC, pr.id ASC;

-- === source: database/dml/query_service/purchase/select_purchases_feed.sql ===
-- name: ListPurchasesFeedFirst :many
-- 指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で先頭ページ取得する。
-- ページを CTE で先に閉じてから結合するのは、明細の要約を解決する LATERAL が LIMIT 前の候補行すべてに
-- 対して評価されるのを防ぐため。
-- 明細 1 件以上は Purchase 集約の生成不変条件（docs/spec/purchase/domain.md）のため、LATERAL は INNER で結合する。
-- filter_by_period=true の場合は注文日時が半開区間 [ordered_after, ordered_before) の購入だけを返す。
WITH page AS (
    SELECT
        p.id,
        p.code,
        p.total_amount,
        p.ordered_at,
        p.status_id
    FROM purchases AS p
    WHERE p.user_id = sqlc.arg('user_id')
        AND (
            NOT sqlc.arg('filter_by_period')::BOOLEAN
            OR (
                p.ordered_at >= sqlc.narg('ordered_after')
                AND p.ordered_at < sqlc.narg('ordered_before')
            )
        )
    ORDER BY p.ordered_at DESC, p.id DESC
    LIMIT sqlc.arg('limit_param')
)

SELECT
    page.id,
    page.code,
    page.total_amount,
    page.ordered_at,
    ps.id AS status_id,
    ps.code AS status_code,
    ps.name AS status_name,
    first_item.product_name AS first_item_name,
    item_agg.item_count
FROM page
INNER JOIN purchase_statuses AS ps ON page.status_id = ps.id
INNER JOIN LATERAL (
    SELECT pr.name AS product_name
    FROM purchase_details AS d
    INNER JOIN products AS pr ON d.product_id = pr.id
    WHERE d.purchase_id = page.id
    ORDER BY d.id
    LIMIT 1
) AS first_item ON TRUE
INNER JOIN LATERAL (
    SELECT COUNT(*)::BIGINT AS item_count
    FROM purchase_details AS d
    WHERE d.purchase_id = page.id
) AS item_agg ON TRUE
ORDER BY page.ordered_at DESC, page.id DESC;

-- name: ListPurchasesFeedAfter :many
-- (ordered_at DESC, id DESC) の keyset 境界より過去の購入履歴を返す。境界は直前ページ末尾行の
-- (ordered_at, id) で、ordered_at 同値は id で安定にタイブレークする。
-- 期間の絞り込みは先頭ページと同一条件で、ページ送りの間も呼び出し側が同じ期間を渡す前提である。
-- ページを CTE で閉じてから要約を結合する形も先頭ページと同一。
WITH page AS (
    SELECT
        p.id,
        p.code,
        p.total_amount,
        p.ordered_at,
        p.status_id
    FROM purchases AS p
    WHERE p.user_id = sqlc.arg('user_id')
        AND (
            p.ordered_at < sqlc.arg('after_ordered_at')
            OR (p.ordered_at = sqlc.arg('after_ordered_at') AND p.id < sqlc.arg('after_id'))
        )
        AND (
            NOT sqlc.arg('filter_by_period')::BOOLEAN
            OR (
                p.ordered_at >= sqlc.narg('ordered_after')
                AND p.ordered_at < sqlc.narg('ordered_before')
            )
        )
    ORDER BY p.ordered_at DESC, p.id DESC
    LIMIT sqlc.arg('limit_param')
)

SELECT
    page.id,
    page.code,
    page.total_amount,
    page.ordered_at,
    ps.id AS status_id,
    ps.code AS status_code,
    ps.name AS status_name,
    first_item.product_name AS first_item_name,
    item_agg.item_count
FROM page
INNER JOIN purchase_statuses AS ps ON page.status_id = ps.id
INNER JOIN LATERAL (
    SELECT pr.name AS product_name
    FROM purchase_details AS d
    INNER JOIN products AS pr ON d.product_id = pr.id
    WHERE d.purchase_id = page.id
    ORDER BY d.id
    LIMIT 1
) AS first_item ON TRUE
INNER JOIN LATERAL (
    SELECT COUNT(*)::BIGINT AS item_count
    FROM purchase_details AS d
    WHERE d.purchase_id = page.id
) AS item_agg ON TRUE
ORDER BY page.ordered_at DESC, page.id DESC;
