-- name: CreateInquiry :execrows
-- 問い合わせを新規登録する。利用者の問い合わせが既にあれば何もせず 0 行を返す
-- （inquiries_user_id_unique。最初の投稿が並行したときに片方が当たる）。
-- UNIQUE 違反を送出しないので、呼び出し側は同じトランザクションのまま先に作られた行を読み直せる。
INSERT INTO inquiries (
    id,
    user_id
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('user_id')
)
ON CONFLICT ON CONSTRAINT inquiries_user_id_unique DO NOTHING;
