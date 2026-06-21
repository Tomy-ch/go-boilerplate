CREATE TABLE IF NOT EXISTS idempotency_keys (
    id UUID NOT NULL DEFAULT GEN_RANDOM_UUID(),
    scope TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_method TEXT NOT NULL,
    request_path TEXT NOT NULL,
    request_fingerprint BYTEA NOT NULL,
    status TEXT NOT NULL,
    response_status INT,
    response_payload BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT idempotency_keys_id_primary PRIMARY KEY (id),
    CONSTRAINT idempotency_keys_scope_key_unique UNIQUE (scope, idempotency_key),
    CONSTRAINT idempotency_keys_status_check CHECK (status IN ('claimed', 'completed'))
);

COMMENT ON TABLE idempotency_keys IS '冪等性キー';
COMMENT ON COLUMN idempotency_keys.id IS 'ID';
COMMENT ON COLUMN idempotency_keys.scope IS 'スコープ（認証プリンシパルID）';
COMMENT ON COLUMN idempotency_keys.idempotency_key IS '冪等性キー（クライアント供給）';
COMMENT ON COLUMN idempotency_keys.request_method IS 'リクエストメソッド';
COMMENT ON COLUMN idempotency_keys.request_path IS 'リクエストパス';
COMMENT ON COLUMN idempotency_keys.request_fingerprint IS 'リクエスト指紋（SHA-256）';
COMMENT ON COLUMN idempotency_keys.status IS '状態（claimed / completed）';
COMMENT ON COLUMN idempotency_keys.response_status IS 'レスポンスHTTPステータス（completedまでNULL）';
COMMENT ON COLUMN idempotency_keys.response_payload IS 'レスポンスペイロード（結果DTOのJSONシリアライズ、completedまでNULL）';
COMMENT ON COLUMN idempotency_keys.created_at IS '作成日時';
COMMENT ON COLUMN idempotency_keys.completed_at IS '完了日時';
COMMENT ON COLUMN idempotency_keys.expires_at IS '有効期限（TTL）';

CREATE INDEX idempotency_keys_expires_at_idx ON idempotency_keys (expires_at);
