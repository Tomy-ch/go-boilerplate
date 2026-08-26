-- name: SummarizePurchasesByUserID :many
-- 指定ユーザーの購入をステータス単位に集計し、購入ステータスマスタの表示順（sort_key 昇順）で返します。
-- 所有権は user_id の等値条件で閉じるため、他ユーザーの購入は集計に混入しません。
-- 既存の複合インデックス purchases (user_id, ordered_at DESC, id DESC) を使う。境界を 1 つも指定しない
-- ときは先頭列（user_id）のみが絞り込みに効きます。
-- キャンセル済み（canceled_at 設定済み）の購入は除外します。
-- Purchase.IsCanceled と同値（database/dml/query_service/README.md 参照）。
-- 注文日時は半開区間 [ordered_after, ordered_before)（internal/usecase/tools/timewindow/README.md）で絞り込みます。
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
        p.ordered_at >= sqlc.narg('ordered_after')
        OR sqlc.narg('ordered_after') IS NULL
    )
    AND (
        p.ordered_at < sqlc.narg('ordered_before')
        OR sqlc.narg('ordered_before') IS NULL
    )
GROUP BY ps.id, ps.code, ps.name, ps.sort_key
ORDER BY ps.sort_key ASC;

-- name: SumPurchaseItemsByUserID :one
-- 指定ユーザーの購入明細の金額合計（単価 × 数量の総和）を価格スケールの正確な decimal で返します。
-- 決済スケール（セント整数）へは丸めません（丸めは決済確定の 1 箇所のみ・ADR-0038）。
-- 母集団は SummarizePurchasesByUserID と同一（所有権・キャンセル除外・期間）です。
-- グループ化を要求されない既定の呼び出しはこのクエリだけで済み、商品単位の行を読みません。
-- 対象が 0 件のとき SUM は NULL を返すため、COALESCE でゼロ値へ畳み込みます。
SELECT COALESCE(SUM(pd.unit_price * pd.quantity), 0)::NUMERIC AS items_total
FROM purchase_details AS pd
INNER JOIN purchases AS p ON pd.purchase_id = p.id
WHERE p.user_id = sqlc.arg('user_id')
    AND p.canceled_at IS NULL
    AND (
        p.ordered_at >= sqlc.narg('ordered_after')
        OR sqlc.narg('ordered_after') IS NULL
    )
    AND (
        p.ordered_at < sqlc.narg('ordered_before')
        OR sqlc.narg('ordered_before') IS NULL
    );

-- name: SummarizePurchaseItemsByProductByUserID :many
-- 指定ユーザーの購入明細を商品単位に集計し、商品が属するカテゴリを添えて返します。
-- 商品は必ず 1 カテゴリに属する（products.category_id は NOT NULL）ため、行は商品単位で一意です。
-- カテゴリ単位だけを要求された場合も、呼び出し側はこの行をカテゴリで畳み込めば得られます。
-- 金額は価格スケールの正確な decimal です（丸めの扱いは SumPurchaseItemsByUserID と同じ）。
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
        p.ordered_at >= sqlc.narg('ordered_after')
        OR sqlc.narg('ordered_after') IS NULL
    )
    AND (
        p.ordered_at < sqlc.narg('ordered_before')
        OR sqlc.narg('ordered_before') IS NULL
    )
GROUP BY pc.id, pc.name, pc.sort_key, pr.id, pr.name
ORDER BY pc.sort_key ASC, pr.name ASC, pr.id ASC;
