-- name: ListShippablePurchases :many
-- 発送可能な購入を、注文日時の古い順（同時刻は ID 昇順）で最大 limit 件取得する。
-- 現在状態は購入ステータスマスタとの結合で code を解決する（status_id は SoT、code は集約が
-- 状態機械の判定に用いる業務キー。JOIN の許容範囲は
-- internal/infrastructure/rdb/repository/README.md の Reference-master exception）。
-- 「発送可能」を定義するのは Purchase.IsShippable で、以下の条件はその実行形です。片方だけ変更しないこと。
-- 支払い済みを表す code は呼び出し側がドメイン定数から渡す。
SELECT
    ps.code AS status_code,
    sqlc.embed(p)
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE ps.code = sqlc.arg('status_code')
ORDER BY p.ordered_at ASC, p.id ASC
LIMIT sqlc.arg('limit_param');

-- name: ListPurchaseDetailsByPurchaseIDs :many
-- 複数の購入 ID から明細をまとめて取得する。購入 1 件ずつの取得を件数分繰り返さないための一括版で、
-- 並びは購入 ID 昇順・同一購入内は明細 ID 昇順。purchase_ids が空の場合は 0 行。
SELECT sqlc.embed(d)
FROM purchase_details AS d
WHERE d.purchase_id = ANY(sqlc.arg('purchase_ids')::UUID[])
ORDER BY d.purchase_id, d.id;
