CREATE TABLE IF NOT EXISTS purchase_details (
    id UUID NOT NULL,
    purchase_id UUID NOT NULL,
    product_id UUID NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price NUMERIC NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
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

-- 購入を起点に明細を辿るための索引（FK 制約は参照列に索引を張らないため明示的に追加する）。
-- 明細を集計する購入集計 API は purchase_details から purchases へ結合するため、この索引が無いと
-- 1 ユーザー分の集計でもテーブル全体を走査することになり、コストが自分の明細数ではなく
-- プラットフォーム全体の明細数に比例する。
-- 商品側（product_id）は products の主キー索引で引けるため、ここでは張らない。
CREATE INDEX purchase_details_purchase_id_idx ON purchase_details (purchase_id);
