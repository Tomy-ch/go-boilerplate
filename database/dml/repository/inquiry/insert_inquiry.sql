-- name: CreateInquiry :exec
-- 問い合わせを新規登録する。利用者の一意制約違反は呼び出し側が衝突として扱う
-- （inquiries_user_id_unique。最初の投稿が並行したときに片方が当たる）。
INSERT INTO inquiries (
    id,
    user_id
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('user_id')
);
