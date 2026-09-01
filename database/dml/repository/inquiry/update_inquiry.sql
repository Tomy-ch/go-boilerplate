-- name: TouchInquiry :exec
-- 最後にメッセージが追加された日時を進める。単調性の検証はドメインが行う。
UPDATE inquiries
SET updated_at = sqlc.arg('updated_at')
WHERE id = sqlc.arg('id');
