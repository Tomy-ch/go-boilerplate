
-- === source: database/dml/system_cqrs/realtime/allocate_stream_sequence.sql ===
-- name: AllocateStreamSequence :one
-- 業務 tx 内でストリームの次の位置を採番する。行が無ければ 1 から始める。
-- UPDATE が取る行ロックは呼び出し側 tx の commit まで保持されるため、同一ストリームの採番は直列化される。
INSERT INTO realtime_stream_sequences (
    stream_id,
    last_sequence
) VALUES (
    $1, 1
)
ON CONFLICT (stream_id) DO UPDATE
    SET last_sequence = realtime_stream_sequences.last_sequence + 1
RETURNING last_sequence;

-- === source: database/dml/system_cqrs/realtime/current_stream_sequence.sql ===
-- name: CurrentStreamSequence :one
-- ストリームの現在位置を返す（History の cursor 用）。未採番のストリームは 0 行を返す。
SELECT last_sequence
FROM realtime_stream_sequences
WHERE stream_id = $1;
