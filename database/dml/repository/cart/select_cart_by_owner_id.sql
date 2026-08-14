-- name: GetCartByOwnerID :one
-- 所有者からカートを 1 件取得する。ユーザー 1 人につきカートは高々 1 件（carts_user_id_unique）。
-- 存在しない場合は 0 行（NotFound）。
-- 有効期限で絞らないのは意図的で、期限切れかどうかの判定はドメインの述語が持つ。ここで取り除くと
-- 取り除かれた行は結果に現れず、不在を観測できないため、突き合わせ検証の余地が消える。
SELECT sqlc.embed(c)
FROM carts AS c
WHERE c.user_id = sqlc.arg('user_id');
