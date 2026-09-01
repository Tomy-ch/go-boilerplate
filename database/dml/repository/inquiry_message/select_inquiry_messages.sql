-- name: ListInquiryMessages :many
-- 問い合わせのメッセージを sequence 昇順で取得する。
-- after_sequence より大きく up_to_sequence 以下の行に限るのが要点で、上限は usecase が先に読んだ
-- stream の現在位置である。上限を掛けることで「現在位置と同じ snapshot で読んだ」のと等価になる
-- （採番の行ロックが commit まで保たれるため、up_to 以下の行は必ず commit 済み）。
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
