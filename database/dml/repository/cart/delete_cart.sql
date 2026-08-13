-- name: DeleteCart :exec
-- カートを削除する。明細は外部キーの連鎖削除で除かれる。存在しない場合もエラーとしない。
DELETE FROM carts
WHERE carts.id = sqlc.arg('id');
