-- name: GetInquiryByUserID :one
-- 利用者の問い合わせを 1 件取得する。利用者 1 人につき高々 1 件（inquiries_user_id_unique）。
-- 存在しない場合は 0 行（NotFound）。
SELECT sqlc.embed(i)
FROM inquiries AS i
WHERE i.user_id = sqlc.arg('user_id');
