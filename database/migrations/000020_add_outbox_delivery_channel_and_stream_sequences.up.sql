-- delivery_channel は relay の claim レーンを分ける。既存行の backfill にのみ一時 DEFAULT を用い、
-- 同一マイグレーション内で落とす。default を残すと channel の指定漏れが無言で HTTP へ流れる。
ALTER TABLE outbox
ADD COLUMN delivery_channel TEXT NOT NULL DEFAULT 'http',
ADD COLUMN ordering_key TEXT,
ADD COLUMN ordering_sequence BIGINT,
ADD COLUMN next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE outbox ALTER COLUMN delivery_channel DROP DEFAULT;

ALTER TABLE outbox
ADD CONSTRAINT outbox_delivery_channel_check
CHECK (delivery_channel IN ('http', 'realtime')),
ADD CONSTRAINT outbox_ordering_pair_check
CHECK ((ordering_key IS NULL) = (ordering_sequence IS NULL));

COMMENT ON COLUMN outbox.delivery_channel IS '配送チャネル（http / realtime）。relay は 1 チャネルのみを claim する';
COMMENT ON COLUMN outbox.ordering_key IS '順序保証の単位（ストリーム）。順序を持たないチャネルは NULL';
COMMENT ON COLUMN outbox.ordering_sequence IS 'ordering_key 内の位置。順序を持たないチャネルは NULL';
COMMENT ON COLUMN outbox.next_attempt_at IS '次に claim してよい時刻（再試行のバックオフ）';

-- claim は channel 内を id 昇順で走査するため、旧 (id) 部分索引を (delivery_channel, id) へ張り替える。
DROP INDEX outbox_pending_idx;

CREATE INDEX outbox_pending_claim_idx ON outbox (delivery_channel, id) WHERE status = 'pending';

-- 先行 sequence の未 published 判定（head-of-line）と、先頭が dead のストリーム集計が引く。
CREATE INDEX outbox_ordering_head_idx ON outbox (ordering_key, ordering_sequence)
WHERE ordering_key IS NOT NULL AND status <> 'published';

-- 同一ストリームへの二重採番を DB で止める。連続 prefix の不変条件を守る最後の砦。
CREATE UNIQUE INDEX outbox_ordering_unique_idx ON outbox (ordering_key, ordering_sequence)
WHERE ordering_key IS NOT NULL;

-- ストリームの採番元。業務トランザクション内で UPDATE ... RETURNING し、行ロックを commit まで保持する。
CREATE TABLE IF NOT EXISTS realtime_stream_sequences (
    stream_id TEXT NOT NULL,
    last_sequence BIGINT NOT NULL,
    CONSTRAINT realtime_stream_sequences_stream_id_primary PRIMARY KEY (stream_id),
    CONSTRAINT realtime_stream_sequences_last_sequence_check CHECK (last_sequence > 0)
);

COMMENT ON TABLE realtime_stream_sequences IS 'ストリーム採番（1 ストリーム 1 行、gap なし単調増加）';
COMMENT ON COLUMN realtime_stream_sequences.stream_id IS 'ストリーム識別子';
COMMENT ON COLUMN realtime_stream_sequences.last_sequence IS '採番済みの最後の位置（1 起算）';
