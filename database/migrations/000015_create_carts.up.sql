CREATE TABLE IF NOT EXISTS carts (
    id UUID NOT NULL,
    user_id UUID,
    session_token TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT carts_id_primary PRIMARY KEY (id),
    CONSTRAINT carts_user_id_foreign FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT carts_user_id_unique UNIQUE (user_id),
    CONSTRAINT carts_session_token_unique UNIQUE (session_token),
    CONSTRAINT carts_owner_exclusive CHECK ((user_id IS NULL) <> (session_token IS NULL))
);

CREATE INDEX carts_expires_at_index ON carts (expires_at);

COMMENT ON TABLE carts IS 'カート';
COMMENT ON COLUMN carts.id IS 'ID';
COMMENT ON COLUMN carts.user_id IS '所有者のユーザーID';
COMMENT ON COLUMN carts.session_token IS 'ゲストセッショントークン';
COMMENT ON COLUMN carts.expires_at IS '有効期限';
COMMENT ON COLUMN carts.created_at IS '作成日時';
COMMENT ON COLUMN carts.updated_at IS '更新日時';
