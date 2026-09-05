
-- === source: database/dml/repository/inquiry/insert_inquiry.sql ===
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

-- === source: database/dml/repository/inquiry/select_inquiries_for_operator.sql ===
-- name: ListInquiriesForOperatorFirst :many
-- 運営向けに問い合わせを (updated_at DESC, id DESC) の安定順で先頭ページ取得する。
SELECT sqlc.embed(i)
FROM inquiries AS i
ORDER BY i.updated_at DESC, i.id DESC
LIMIT sqlc.arg('page_size');

-- name: ListInquiriesForOperatorAfter :many
-- 同上を cursor（updated_at, id）より後ろから取得する。複合 cursor の比較は行値式で行い、
-- updated_at が同値の行を id で決着させる（先頭ページと同じ全順序を保つ）。
SELECT sqlc.embed(i)
FROM inquiries AS i
-- 2 番目の要素だけ明示的にキャストする。キャストが無いと sqlc は行値式の 2 要素目の型を推論できず、
-- 1 要素目（timestamptz）を流用した引数を生成する。1 要素目にもキャストを付けると
-- sqlc.yaml の pg_catalog.timestamptz 上書きから外れ pgtype へ落ちるため、こちらは付けない。
WHERE (i.updated_at, i.id) < (sqlc.arg('cursor_updated_at'), sqlc.arg('cursor_id')::UUID)
ORDER BY i.updated_at DESC, i.id DESC
LIMIT sqlc.arg('page_size');

-- === source: database/dml/repository/inquiry/select_inquiry_by_id.sql ===
-- name: GetInquiryByID :one
-- 問い合わせを 1 件取得する。存在しない場合は 0 行（NotFound）。
SELECT sqlc.embed(i)
FROM inquiries AS i
WHERE i.id = sqlc.arg('id');

-- === source: database/dml/repository/inquiry/select_inquiry_by_user_id.sql ===
-- name: GetInquiryByUserID :one
-- 利用者の問い合わせを 1 件取得する。利用者 1 人につき高々 1 件（inquiries_user_id_unique）。
-- 存在しない場合は 0 行（NotFound）。
SELECT sqlc.embed(i)
FROM inquiries AS i
WHERE i.user_id = sqlc.arg('user_id');

-- === source: database/dml/repository/inquiry/update_inquiry.sql ===
-- name: TouchInquiry :exec
-- 最後にメッセージが追加された日時を進める。単調性の検証はドメインが行う。
UPDATE inquiries
SET updated_at = sqlc.arg('updated_at')
WHERE id = sqlc.arg('id');
