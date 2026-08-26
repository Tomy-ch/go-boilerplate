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
    lock_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT products_id_primary PRIMARY KEY (id),
    CONSTRAINT products_status_id_foreign FOREIGN KEY (status_id) REFERENCES product_statuses (id),
    CONSTRAINT products_category_id_foreign FOREIGN KEY (category_id) REFERENCES product_categories (id)
);

-- 公開商品一覧を keyset ページネーションするための複合インデックス。
-- WHERE published_at IS NOT NULL ORDER BY published_at DESC, id DESC を index range scan で処理する。
-- 逆順の走査で昇順（sort=publishedAt）も同じ索引で賄えるため、並び順ごとには分けない。
-- price / quantity の範囲条件は INCLUDE 列から評価し、絞り込みで捨てる行の heap 参照を避ける。
CREATE INDEX products_published_at_id_idx
ON products (published_at DESC, id DESC)
INCLUDE (price, quantity)
WHERE published_at IS NOT NULL;

-- 一覧の絞り込みで使う参照列（FK 制約は参照列に索引を張らないため明示的に追加する）。
CREATE INDEX products_category_id_idx ON products (category_id);
CREATE INDEX products_status_id_idx ON products (status_id);

-- 在庫僅少一覧（ORDER BY quantity ASC, id ASC）のためのインデックス。
-- 閾値未設定の商品はこの一覧に現れないため、部分インデックスで除外する。
CREATE INDEX products_low_stock_quantity_id_idx
ON products (quantity ASC, id ASC)
WHERE stock_warning_threshold IS NOT NULL;

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
COMMENT ON COLUMN products.lock_version IS '楽観ロックバージョン';
COMMENT ON COLUMN products.created_at IS '作成日時';
COMMENT ON COLUMN products.updated_at IS '更新日時';
