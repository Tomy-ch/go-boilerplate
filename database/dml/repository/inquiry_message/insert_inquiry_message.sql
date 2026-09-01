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
