-- name: ListInquiryMessages :many
-- 問い合わせのメッセージを stream_sequence 昇順で取得する。
-- after_sequence より大きく up_to_sequence 以下の行に限ること（上限は usecase が先に読んだ stream の
-- 現在位置。論拠は docs/spec/inquiry/usecase.md の「streamCursor と snapshot」）。
SELECT sqlc.embed(m)
FROM inquiry_messages AS m
WHERE m.inquiry_id = sqlc.arg('inquiry_id')
    AND (
        m.stream_sequence > sqlc.narg('after_sequence')
        OR sqlc.narg('after_sequence') IS NULL
    )
    AND m.stream_sequence <= sqlc.arg('up_to_sequence')
ORDER BY m.stream_sequence ASC
LIMIT sqlc.arg('page_size');
