DROP TABLE IF EXISTS realtime_stream_sequences;

DROP INDEX IF EXISTS outbox_ordering_unique_idx;

DROP INDEX IF EXISTS outbox_ordering_head_idx;

DROP INDEX IF EXISTS outbox_pending_claim_idx;

CREATE INDEX outbox_pending_idx ON outbox (id) WHERE status = 'pending';

ALTER TABLE outbox
DROP CONSTRAINT IF EXISTS outbox_ordering_pair_check,
DROP CONSTRAINT IF EXISTS outbox_delivery_channel_check;

ALTER TABLE outbox
DROP COLUMN IF EXISTS next_attempt_at,
DROP COLUMN IF EXISTS ordering_sequence,
DROP COLUMN IF EXISTS ordering_key,
DROP COLUMN IF EXISTS delivery_channel;
