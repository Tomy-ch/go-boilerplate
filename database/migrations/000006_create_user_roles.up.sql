CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_roles_primary PRIMARY KEY (user_id, role_id),
    CONSTRAINT user_roles_user_id_foreign FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT user_roles_role_id_foreign FOREIGN KEY (role_id) REFERENCES roles (id)
);

COMMENT ON TABLE user_roles IS 'ユーザロール';
COMMENT ON COLUMN user_roles.user_id IS 'ユーザID';
COMMENT ON COLUMN user_roles.role_id IS 'ロールID';
COMMENT ON COLUMN user_roles.created_at IS '作成日時';
COMMENT ON COLUMN user_roles.updated_at IS '更新日時';
