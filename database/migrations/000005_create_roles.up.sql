CREATE TABLE IF NOT EXISTS roles (
    id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    code SMALLINT NOT NULL,
    sort_key SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT roles_id_primary PRIMARY KEY (id),
    CONSTRAINT roles_name_unique UNIQUE (name),
    CONSTRAINT roles_code_unique UNIQUE (code),
    CONSTRAINT roles_sort_key_unique UNIQUE (sort_key)
);

COMMENT ON TABLE roles IS 'ロール';
COMMENT ON COLUMN roles.id IS 'ID';
COMMENT ON COLUMN roles.name IS '名称';
COMMENT ON COLUMN roles.code IS 'コード';
COMMENT ON COLUMN roles.sort_key IS '順序';
COMMENT ON COLUMN roles.created_at IS '作成日時';
COMMENT ON COLUMN roles.updated_at IS '更新日時';

INSERT INTO roles (id, name, code, sort_key) VALUES
('a1b2c3d4-0000-4000-8000-000000000001', '管理者', 1, 1) ON CONFLICT (id) DO NOTHING;
INSERT INTO roles (id, name, code, sort_key) VALUES
('a1b2c3d4-0000-4000-8000-000000000002', '一般', 2, 2) ON CONFLICT (id) DO NOTHING;
