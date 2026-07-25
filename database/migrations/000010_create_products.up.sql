CREATE TABLE IF NOT EXISTS products (
    id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price NUMERIC NOT NULL,
    quantity INTEGER NOT NULL,
    stock_warning_threshold INTEGER,
    status_id UUID NOT NULL,
    category_id UUID NOT NULL,
    published_at TIMESTAMPTZ,
    image_path TEXT,
    lock_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT products_id_primary PRIMARY KEY (id),
    CONSTRAINT products_status_id_foreign FOREIGN KEY (status_id) REFERENCES product_statuses (id),
    CONSTRAINT products_category_id_foreign FOREIGN KEY (category_id) REFERENCES product_categories (id)
);

COMMENT ON TABLE products IS '商品';
COMMENT ON COLUMN products.id IS 'ID';
COMMENT ON COLUMN products.name IS '名称';
COMMENT ON COLUMN products.description IS '説明';
COMMENT ON COLUMN products.price IS '価格';
COMMENT ON COLUMN products.quantity IS '在庫数';
COMMENT ON COLUMN products.stock_warning_threshold IS '在庫警告閾値';
COMMENT ON COLUMN products.status_id IS '商品ステータスID';
COMMENT ON COLUMN products.category_id IS '商品カテゴリID';
COMMENT ON COLUMN products.published_at IS '公開日時';
COMMENT ON COLUMN products.image_path IS '画像パス';
COMMENT ON COLUMN products.lock_version IS '楽観ロックバージョン';
COMMENT ON COLUMN products.created_at IS '作成日時';
COMMENT ON COLUMN products.updated_at IS '更新日時';
