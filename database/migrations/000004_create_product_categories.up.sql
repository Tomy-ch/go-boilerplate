CREATE TABLE IF NOT EXISTS product_categories (
    id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    code SMALLINT NOT NULL,
    sort_key SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT product_categories_id_primary PRIMARY KEY (id),
    CONSTRAINT product_categories_name_unique UNIQUE (name),
    CONSTRAINT product_categories_code_unique UNIQUE (code),
    CONSTRAINT product_categories_sort_key_unique UNIQUE (sort_key)
);

COMMENT ON TABLE product_categories IS '商品カテゴリ';
COMMENT ON COLUMN product_categories.id IS 'ID';
COMMENT ON COLUMN product_categories.name IS '名称';
COMMENT ON COLUMN product_categories.code IS 'コード';
COMMENT ON COLUMN product_categories.sort_key IS '順序';
COMMENT ON COLUMN product_categories.created_at IS '作成日時';
COMMENT ON COLUMN product_categories.updated_at IS '更新日時';

-- Insert initial product categories data
INSERT INTO product_categories (id, name, code, sort_key) VALUES
('5dd52d84-78eb-4a52-ba0b-2e11c95c2af2', '電子機器', 1, 1) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_categories (id, name, code, sort_key) VALUES
('b39be992-fe5a-4b4c-9f98-e695f0f5101e', '書籍', 2, 2) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_categories (id, name, code, sort_key) VALUES
('3a60c501-7049-4a63-bfd3-bf34555f3aec', '衣料品', 3, 3) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_categories (id, name, code, sort_key) VALUES
('fee06f48-4aa6-4e6d-810c-6dfcb970b5f7', '食品', 4, 4) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_categories (id, name, code, sort_key) VALUES
('93e544f8-0815-4b62-bbcd-c02cfc305818', '家具', 5, 5) ON CONFLICT (id) DO NOTHING;
