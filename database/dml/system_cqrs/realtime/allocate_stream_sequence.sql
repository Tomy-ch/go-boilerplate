-- name: AllocateStreamSequence :one
-- 業務 tx 内でストリームの次の位置を採番する。行が無ければ 1 から始める。
-- 同一ストリームの採番は行ロックで直列化される（ADR-0072）。
INSERT INTO realtime_stream_sequences (
    stream_id,
    last_sequence
) VALUES (
    $1, 1
)
ON CONFLICT (stream_id) DO UPDATE
    SET last_sequence = realtime_stream_sequences.last_sequence + 1
RETURNING last_sequence;
