-- name: GetPurchaseByIDForUpdate :one
-- ID から購入を 1 件、購入行のみ悲観ロック（FOR UPDATE OF p）して取得する。キャンセルの状態遷移の
-- 競合（同一購入への並行キャンセル）を購入行ロックで直列化する（結合先の固定参照マスタはロックしない）。
-- 現在状態は購入ステータスマスタとの結合で code を解決する。存在しない場合は 0 行（NotFound）。
SELECT
    ps.code AS status_code,
    sqlc.embed(p)
FROM purchases AS p
INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
WHERE p.id = @id
FOR UPDATE OF p;
