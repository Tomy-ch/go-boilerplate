CREATE TABLE IF NOT EXISTS user_identities (
    id UUID NOT NULL,
    user_id UUID NOT NULL,
    issuer VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_identities_id_primary PRIMARY KEY (id),
    CONSTRAINT user_identities_issuer_subject_unique UNIQUE (issuer, subject),
    CONSTRAINT user_identities_user_id_issuer_unique UNIQUE (user_id, issuer),
    CONSTRAINT user_identities_user_id_foreign FOREIGN KEY (user_id) REFERENCES users (id)
);

COMMENT ON TABLE user_identities IS '外部ID連携';
COMMENT ON COLUMN user_identities.id IS 'ID';
COMMENT ON COLUMN user_identities.user_id IS 'ユーザID';
COMMENT ON COLUMN user_identities.issuer IS 'トークン発行者（IdP issuer）';
COMMENT ON COLUMN user_identities.subject IS '認証主体（token の sub）';
COMMENT ON COLUMN user_identities.created_at IS '作成日時';
COMMENT ON COLUMN user_identities.updated_at IS '更新日時';
