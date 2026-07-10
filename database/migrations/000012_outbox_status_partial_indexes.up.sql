CREATE INDEX outbox_published_gc_idx ON outbox (published_at) WHERE status = 'published';
CREATE INDEX outbox_dead_idx ON outbox (id) WHERE status = 'dead';
