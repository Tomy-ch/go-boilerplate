-- name: CurrentStreamSequence :one
-- ストリームの現在位置を返す（History の cursor 用）。未採番のストリームは 0 行を返す。
SELECT last_sequence
FROM realtime_stream_sequences
WHERE stream_id = $1;
