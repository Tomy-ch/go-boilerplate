
-- === source: database/dml/repository/inquiry_message/insert_inquiry_message.sql ===
-- name: CreateInquiryMessage :exec
-- メッセージを 1 件追加する。(inquiry_id, sequence) の一意制約違反は呼び出し側が衝突として扱う
-- （採番と同一 tx で呼ぶ限り到達しない防御）。
INSERT INTO inquiry_messages (
    id,
    inquiry_id,
    author_kind,
    author_subject_id,
    body,
    sequence
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('inquiry_id'),
    sqlc.arg('author_kind'),
    sqlc.arg('author_subject_id'),
    sqlc.arg('body'),
    sqlc.arg('sequence')
);

-- === source: database/dml/repository/inquiry_message/select_inquiry_messages.sql ===
-- name: ListInquiryMessages :many
-- 問い合わせのメッセージを sequence 昇順で取得する。
-- after_sequence より大きく up_to_sequence 以下の行に限ること（上限は usecase が先に読んだ stream の
-- 現在位置。論拠は docs/spec/inquiry/usecase.md の「streamCursor と snapshot」）。
SELECT sqlc.embed(m)
FROM inquiry_messages AS m
WHERE m.inquiry_id = sqlc.arg('inquiry_id')
AND (
    m.sequence > sqlc.narg('after_sequence')
    OR sqlc.narg('after_sequence') IS NULL
)
AND m.sequence <= sqlc.arg('up_to_sequence')
ORDER BY m.sequence ASC
LIMIT sqlc.arg('page_size');
