CREATE TABLE IF NOT EXISTS cart_items (
    id UUID NOT NULL,
    cart_id UUID NOT NULL,
    product_id UUID NOT NULL,
    quantity INTEGER NOT NULL,
    last_seen_price NUMERIC,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT cart_items_id_primary PRIMARY KEY (id),
    CONSTRAINT cart_items_cart_id_foreign FOREIGN KEY (cart_id) REFERENCES carts (id) ON DELETE CASCADE,
    CONSTRAINT cart_items_product_id_foreign FOREIGN KEY (product_id) REFERENCES products (id),
    CONSTRAINT cart_items_cart_id_product_id_unique UNIQUE (cart_id, product_id),
    CONSTRAINT cart_items_quantity_positive CHECK (quantity >= 1)
);

COMMENT ON TABLE cart_items IS 'カート明細';
COMMENT ON COLUMN cart_items.id IS 'ID';
COMMENT ON COLUMN cart_items.cart_id IS 'カートID';
COMMENT ON COLUMN cart_items.product_id IS '商品ID';
COMMENT ON COLUMN cart_items.quantity IS '数量';
COMMENT ON COLUMN cart_items.last_seen_price IS '最後に提示した価格';
COMMENT ON COLUMN cart_items.added_at IS '追加日時';
COMMENT ON COLUMN cart_items.created_at IS '作成日時';
COMMENT ON COLUMN cart_items.updated_at IS '更新日時';
