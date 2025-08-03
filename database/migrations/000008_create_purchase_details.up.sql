CREATE TABLE IF NOT EXISTS purchase_details (
    id UUID NOT NULL,
    purchase_id UUID NOT NULL,
    product_id UUID NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT purchase_details_id_primary PRIMARY KEY (id),
    CONSTRAINT purchase_details_purchase_id_foreign FOREIGN KEY (purchase_id) REFERENCES purchases (id),
    CONSTRAINT purchase_details_product_id_foreign FOREIGN KEY (product_id) REFERENCES products (id)
);

COMMENT ON TABLE purchase_details IS '購入詳細';
COMMENT ON COLUMN purchase_details.id IS 'ID';
COMMENT ON COLUMN purchase_details.purchase_id IS '購入ID';
COMMENT ON COLUMN purchase_details.product_id IS '商品ID';
COMMENT ON COLUMN purchase_details.quantity IS '数量';
COMMENT ON COLUMN purchase_details.unit_price IS '単価';
COMMENT ON COLUMN purchase_details.created_at IS '作成日時';
COMMENT ON COLUMN purchase_details.updated_at IS '更新日時';
