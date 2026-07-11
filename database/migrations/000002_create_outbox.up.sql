CREATE TABLE IF NOT EXISTS outbox (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    message_id UUID NOT NULL DEFAULT GEN_RANDOM_UUID(),
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    headers JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    CONSTRAINT outbox_id_primary PRIMARY KEY (id),
    CONSTRAINT outbox_message_id_unique UNIQUE (message_id),
    CONSTRAINT outbox_status_check CHECK (status IN ('pending', 'published', 'dead'))
);

COMMENT ON TABLE outbox IS 'トランザクショナル outbox（ドメインイベントの信頼 publish）';
COMMENT ON COLUMN outbox.id IS 'ID';
COMMENT ON COLUMN outbox.message_id IS 'dedup の安定キー（INSERT 時採番、Idempotency-Key へ伝搬）';
COMMENT ON COLUMN outbox.aggregate_type IS '集約種別（観測・調査用。順序キーではない）';
COMMENT ON COLUMN outbox.aggregate_id IS '集約ID（観測・調査用。順序キーではない）';
COMMENT ON COLUMN outbox.event_type IS 'イベント種別 + version';
COMMENT ON COLUMN outbox.payload IS 'ペイロード（snapshot + version の収束可能な自己完結ペイロード）';
COMMENT ON COLUMN outbox.headers IS 'publish 時に伝搬するヘッダ（traceparent 等）';
COMMENT ON COLUMN outbox.status IS '状態（pending / published / dead）';
COMMENT ON COLUMN outbox.attempts IS 'publish 試行回数';
COMMENT ON COLUMN outbox.last_error IS '直近の publish 失敗理由';
COMMENT ON COLUMN outbox.created_at IS '作成日時';
COMMENT ON COLUMN outbox.published_at IS 'publish 完了日時（published 遷移時刻）';

CREATE INDEX outbox_pending_idx ON outbox (id) WHERE status = 'pending';
CREATE INDEX outbox_published_gc_idx ON outbox (published_at) WHERE status = 'published';
CREATE INDEX outbox_dead_idx ON outbox (id) WHERE status = 'dead';
