-- name: GetInquiryByID :one
-- 問い合わせを 1 件取得する。存在しない場合は 0 行（NotFound）。
SELECT sqlc.embed(i)
FROM inquiries AS i
WHERE i.id = sqlc.arg('id');
