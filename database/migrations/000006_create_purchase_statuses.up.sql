CREATE TABLE IF NOT EXISTS purchase_statuses (
    id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    code SMALLINT NOT NULL,
    sort_key SMALLINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT purchase_statuses_id_primary PRIMARY KEY (id),
    CONSTRAINT purchase_statuses_name_unique UNIQUE (name),
    CONSTRAINT purchase_statuses_code_unique UNIQUE (code),
    CONSTRAINT purchase_statuses_sort_key_unique UNIQUE (sort_key)
);

COMMENT ON TABLE purchase_statuses IS '購入ステータス';
COMMENT ON COLUMN purchase_statuses.id IS 'ID';
COMMENT ON COLUMN purchase_statuses.name IS '名称';
COMMENT ON COLUMN purchase_statuses.code IS 'コード';
COMMENT ON COLUMN purchase_statuses.sort_key IS '順序';
COMMENT ON COLUMN purchase_statuses.created_at IS '作成日時';
COMMENT ON COLUMN purchase_statuses.updated_at IS '更新日時';

-- Insert initial purchase statuses data
INSERT INTO purchase_statuses (id, name, code, sort_key) VALUES
    ('a66c996c-86b2-41d8-9bdd-9b685fb7c47d', '未処理', 1, 1) ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_statuses (id, name, code, sort_key) VALUES
    ('e328c9f7-ea8b-49a7-a798-77bc538e3ffe', '受付中', 2, 2) ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_statuses (id, name, code, sort_key) VALUES
    ('c6b37666-ffe3-4969-9de7-a7eb6ccd2d74', '確認中', 3, 3) ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_statuses (id, name, code, sort_key) VALUES
    ('323dee43-6553-4f9d-a55f-086ad5625eef', '処理中', 4, 4) ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_statuses (id, name, code, sort_key) VALUES
    ('1904bf76-7d37-4288-bc15-359d2512ac91', '完了', 5, 5) ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_statuses (id, name, code, sort_key) VALUES
    ('e9d72547-adfe-48d9-9037-bd1f55d4158b', 'キャンセル', 6, 6) ON CONFLICT (id) DO NOTHING;
